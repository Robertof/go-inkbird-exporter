package collector

import (
	"context"
	"fmt"

	"github.com/robertof/go-inkbird-exporter/ble"
	"github.com/robertof/go-inkbird-exporter/collector/model"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/robertof/go-inkbird-exporter/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func connectAndCollect(
	ctx context.Context,
	handle *ble.Handle,
	device deviceWithBackend[device.ActiveBackend],
) (reading device.Reading, err error) {
	conn, err := handle.Connect(ctx, device.Addr())

	if err != nil {
		return reading, fmt.Errorf("failed to connect to device: %w", err)
	}

	reading, err = device.backend.Read(conn)

	if err != nil {
		return reading, fmt.Errorf("failed to read data from device: %w", err)
	}

	return reading, nil
}

func collectViaConnection(
	ctx context.Context,
	handle *ble.Handle,
	devices []deviceWithBackend[device.ActiveBackend],
	ch chan model.DeviceResult,
) error {
	var eg errgroup.Group

	log.Trace().
		Array("Devices", utils.ToZeroLogArray(devices)).
		Msg("collectViaConnection: started")

	for _, device := range devices {
		device := device

		eg.Go(func() error {
			log.Trace().
				Stringer("Device", device).
				Msg("collectViaConnection: device worker started")

			reading, err := connectAndCollect(ctx, handle, device)

			result := model.DeviceResult{
				Device: device.Device,
				Result: model.Result{
					Reading: reading,
					Error: err,
				},
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- result:
			}

			log.Trace().
				Stringer("Device", device).
				Msg("collectViaConnection: device worker finished and submitted work")

			return nil
		})
	}

	return eg.Wait()
}
