package collector

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/robertof/go-inkbird-exporter/ble"
	"github.com/robertof/go-inkbird-exporter/collector/model"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/rs/zerolog/log"
)

func collectViaScan(
	ctx context.Context,
	handle *ble.Handle,
	devices []deviceWithBackend[device.PassiveBackend],
	ch chan model.DeviceResult,
) error {
	type DeviceContext struct {
		deviceWithBackend[device.PassiveBackend]
		sync.Once
	}

	numLeft := len(devices)
  addresses := make([]net.HardwareAddr, len(devices))
  deviceMap := make(map[string]*DeviceContext)

  {
    i := 0
    for _, device := range devices {
      addresses[i] = device.Addr()
      deviceMap[strings.ToLower(device.Addr().String())] = &DeviceContext{
      	deviceWithBackend: device,
      }
      i += 1
    }
  }

  err := handle.ScanAddresses(ctx, addresses, func(a ble.Advertisement) bool {
    deviceCtx := deviceMap[strings.ToLower(a.Addr().String())]

    if deviceCtx == nil {
      log.Warn().
        Str("Address", a.Addr().String()).
        Str("LocalName", a.LocalName()).
        Hex("ManufacturerData", a.ManufacturerData()).
        Interface("ServiceData", a.ServiceData()).
        Msg("Received advertisement from unknown device!")

      return false
    }

    log.Trace().
      Stringer("Device", deviceCtx.Device).
      Str("LocalName", a.LocalName()).
      Hex("ManufacturerData", a.ManufacturerData()).
      Interface("ServiceData", a.ServiceData()).
      Msg("collectViaScan: received advertisement from device")

    reading, err := deviceCtx.backend.ParseAdvertisement(a)

    log.Trace().
      Err(err).
      Stringer("Reading", reading).
      Stringer("Device", deviceCtx.Device).
      Msg("collectViaScan: parsed device advertisement")

		result := model.DeviceResult{
			Device: deviceCtx.Device,
			Result: model.Result{
				Reading: reading,
				Error: err,
			},
		}

		select {
		case <-ctx.Done():
			return true // context is canceled, let's get out of the way
		case ch <- result:
		}

    deviceCtx.Do(func() {
    	numLeft -= 1
    })

    return err == nil // consider ourselves happy when there is no error parsing the advertisement
  })

  // swallow deadline exceeded errors if we got results for all devices
  if errors.Is(err, context.DeadlineExceeded) && numLeft == 0 {
    err = nil
  }

  return err
}
