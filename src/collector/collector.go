package collector

import (
  "context"
  "errors"
  "fmt"
  "net"
  "strings"
  "sync"
  "time"

  "github.com/robertof/go-inkbird-exporter/ble"
  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/robertof/go-inkbird-exporter/utils"
  "github.com/rs/zerolog/log"
)

const (
  DefaultMaxRetries = 2
  DefaultTimeoutPerAttempt = 5 * time.Second
  DefaultBackoffFactor = 500 * time.Millisecond
)

type CollectionOptions struct {
  MaxRetries int
  TimeoutPerAttempt time.Duration
  BackoffFactor time.Duration

  attempt int
}

type CollectionResult struct {
  Reading device.Reading
  Error error
}

func (c CollectionResult) String() string {
  if c.Error != nil {
    return fmt.Sprintf("result:error(%v)", c.Error)
  } else {
    return fmt.Sprintf("result:success(%v)", c.Reading)
  }
}

func newCollectionResult(reading device.Reading, error error) CollectionResult {
  return CollectionResult{
    Reading: reading,
    Error: error,
  }
}

func CollectReadings(
  handle *ble.Handle,
  ctx context.Context,
  devices []device.Device,
) (out map[device.Device]CollectionResult, err error) {
  return CollectReadingsWithOptions(
    handle,
    ctx,
    devices,
    CollectionOptions{
      MaxRetries: DefaultMaxRetries,
      TimeoutPerAttempt: DefaultTimeoutPerAttempt,
    },
  )
}

// Collect readings from the specified devices and don't stop until either all advertisements
// have been parsed successfully or the context timeout (if any) expires.
func CollectReadingsWithOptions(
  handle *ble.Handle,
  parentCtx context.Context,
  devices []device.Device,
  options CollectionOptions,
) (out map[device.Device]CollectionResult, err error) {
  addresses := make([]net.HardwareAddr, len(devices))
  deviceMap := make(map[string]device.Device)
  out = make(map[device.Device]CollectionResult, len(devices))

  var mu sync.Mutex

  log.Debug().
    Array("Devices", utils.ToZeroLogArray(devices)).
    Msg("Collecting readings from devices")

  {
    i := 0
    for _, device := range devices {
      addresses[i] = device.Addr()
      deviceMap[strings.ToLower(device.Addr().String())] = device
    }
  }

  // make sure signals are properly handled and we enforce the passed timeout.
  var ctx context.Context
  var cancel func()

  if options.TimeoutPerAttempt > 0 {
    ctx, cancel = context.WithTimeout(parentCtx, options.TimeoutPerAttempt)
  } else {
    ctx, cancel = context.WithCancel(parentCtx)
  }

  ctx = ble.WrapContextWithSigHandler(ctx, cancel)

  err = handle.ScanAddresses(ctx, addresses, func(a ble.Advertisement) bool {
    device := deviceMap[strings.ToLower(a.Addr().String())]

    if device == nil {
      log.Warn().
        Str("Address", a.Addr().String()).
        Str("LocalName", a.LocalName()).
        Hex("ManufacturerData", a.ManufacturerData()).
        Interface("ServiceData", a.ServiceData()).
        Msg("Received advertisement from unknown device!")

      return false
    }

    log.Trace().
      Stringer("Device", device).
      Str("LocalName", a.LocalName()).
      Hex("ManufacturerData", a.ManufacturerData()).
      Interface("ServiceData", a.ServiceData()).
      Msg("Received advertisement from device")

    reading, err := device.ParseAdvertisement(a)

    log.Trace().
      Err(err).
      Stringer("Reading", reading).
      Stringer("Device", device).
      Msg("Parsed device advertisement")

    result := newCollectionResult(reading, err)

    // this might be called concurrently if multiple advertisements come at once.
    mu.Lock()
    defer mu.Unlock()

    out[device] = result

    return err == nil // consider ourselves happy when there is no error parsing the advertisement
  })

  // swallow deadline exceeded errors if we got results for all devices
  if errors.Is(err, context.DeadlineExceeded) && len(out) == len(devices) {
    err = nil
  }

  // analyze results, and retry if needed
  if options.MaxRetries > 0 {
    var failedDevices []device.Device

    for _, device := range devices {
      if result, ok := out[device]; ok && result.Error != nil {
        // parsing failed
        failedDevices = append(failedDevices, device)

        log.Debug().
          Stringer("Device", device).
          Int("RetriesLeft", options.MaxRetries).
          Err(result.Error).
          Msg("Collection failed for device - will retry")
      } else if !ok {
        // never got a result for the device
        failedDevices = append(failedDevices, device)

        log.Debug().
          Stringer("Device", device).
          Int("RetriesLeft", options.MaxRetries).
          Err(err).
          Msg("No data received for device (wrong MAC?) - will retry")
      }
    }

    if len(failedDevices) > 0 {
      if options.BackoffFactor > 0 {
        backoff := options.BackoffFactor << int64(options.attempt)

        if backoff < 0 {
          backoff = DefaultBackoffFactor
        }

        log.Trace().
          Dur("Backoff", backoff).
          Msg("Backing off before attempting retry")

        select {
        case <-parentCtx.Done():
          log.Trace().Err(ctx.Err()).Msg("Retry aborted by context cancel")
          return out, ctx.Err()
        case <-time.After(backoff):
        }
      }

      options.MaxRetries -= 1
      options.attempt += 1

      retryOutput, err := CollectReadingsWithOptions(handle, parentCtx, failedDevices, options)

      // merge old and new outputs
      if retryOutput != nil {
        for failedDevice := range retryOutput {
          out[failedDevice] = retryOutput[failedDevice]
        }
      }

      if err != nil {
        return out, err
      }

      return out, nil
    }
  }

  return out, err
}
