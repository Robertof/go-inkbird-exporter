package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/robertof/go-inkbird-exporter/ble"
	"github.com/robertof/go-inkbird-exporter/collector/model"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/robertof/go-inkbird-exporter/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
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

type deviceWithBackend[Backend any] struct {
  device.Device
  backend Backend
}

func selectDevicesByBackend(devices []device.Device) (
  passive []deviceWithBackend[device.PassiveBackend],
  active []deviceWithBackend[device.ActiveBackend],
) {
  for _, dev := range devices {
    switch backend := dev.Backend().(type) {
    case device.ActiveBackend:
      active = append(active, deviceWithBackend[device.ActiveBackend]{
        Device: dev,
        backend: backend,
      })
    case device.PassiveBackend:
      passive = append(passive, deviceWithBackend[device.PassiveBackend]{
        Device: dev,
        backend: backend,
      })
    default:
      panic(fmt.Sprintf(
        "device %q has invalid backend %q, must be one of ActiveBackend or PassiveBackend",
        dev,
        backend,
      ))
    }
  }

  return passive, active
}

func CollectReadings(
  handle *ble.Handle,
  ctx context.Context,
  devices []device.Device,
) (out map[device.Device]model.Result, err error) {
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
) (out map[device.Device]model.Result, err error) {
  out = make(map[device.Device]model.Result, len(devices))

  log.Debug().
    Array("Devices", utils.ToZeroLogArray(devices)).
    Msg("Collecting readings from devices")

  passiveDevices, activeDevices := selectDevicesByBackend(devices)

  // make sure signals are properly handled and we enforce the passed timeout.
  var ctx context.Context
  var cancel func()

  if options.TimeoutPerAttempt > 0 {
    ctx, cancel = context.WithTimeout(parentCtx, options.TimeoutPerAttempt)
  } else {
    ctx, cancel = context.WithCancel(parentCtx)
  }

  defer cancel()

  // collect everything in parallel and gather results.
  var eg errgroup.Group
  resultCh := make(chan model.DeviceResult)

  if len(passiveDevices) > 0 {
    log.Trace().
      Array("Devices", utils.ToZeroLogArray(passiveDevices)).
      Msg("Collecting data from devices via scan")
    eg.Go(func() error {
      return collectViaScan(ctx, handle, passiveDevices, resultCh)
    })
  }

  if len(activeDevices) > 0 {
    log.Trace().
      Array("Devices", utils.ToZeroLogArray(activeDevices)).
      Msg("Collecting data from devices via direct connection")
    eg.Go(func() error {
      return collectViaConnection(ctx, handle, activeDevices, resultCh)
    })
  }

  go func() {
    err = eg.Wait()
    close(resultCh)
  }()

  for v := range resultCh {
    log.Trace().
      Stringer("Device", v.Device).
      Stringer("Result", v.Result).
      Msg("Received result for device")

    out[v.Device] = v.Result
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
