package main

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"

	"github.com/robertof/go-inkbird-exporter/ble"
)

func doDeviceDiscovery(cfg config) {
  log.Info().Msg("Starting in device discovery mode - collecting devices for 5 seconds...")

  handle, err := ble.Init(cfg.BluetoothDeviceId, ble.FlagScanTypeActive)

  if err != nil {
    log.Fatal().Err(err).Msg("Failed to initialize Bluetooth device")
  }

  ctx := ble.WrapContextWithSigHandler(
    context.WithTimeout(
      context.Background(),
      5 * time.Second,
    ),
  )

  type deviceInfo struct {
    name string
    connectable bool
    services []string
  }

  devices := make(map[string]deviceInfo)

  err = handle.ScanAll(ctx, func(a ble.Advertisement) {
    services := make(map[string]bool)

    for _, uuid := range a.Services() {
      services[uuid.String()] = true
    }

    var info deviceInfo
    var ok bool

    if info, ok = devices[a.Addr().String()]; ok {
      // merge
      if info.name == "" {
        info.name = a.LocalName()
      }
      info.connectable = a.Connectable()

      for _, uuid := range info.services {
        if _, ok := services[uuid]; !ok {
          services[uuid] = true
        }
      }

      info.services = maps.Keys(services)
    } else {
      info = deviceInfo{
        name: a.LocalName(),
        connectable: a.Connectable(),
        services: maps.Keys(services),
      }
    }

    devices[a.Addr().String()] = info

    log.Debug().
      Str("Addr", a.Addr().String()).
      Str("Name", a.LocalName()).
      Bool("Connectable", a.Connectable()).
      Strs("Services", maps.Keys(services)).
      Hex("ManufacturerData", a.ManufacturerData()).
      Msg("Received device advertisement")
  })

  if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
    log.Fatal().Err(err).Msg("Failed to initiate scan")
  }

  log.Info().Int("Found", len(devices)).Msg("Finished device discovery")

  for addr, data := range devices {
    log.Info().
      Str("Addr", addr).
      Str("Name", data.name).
      Bool("Connectable", data.connectable).
      Strs("Services", data.services).
      Msg("Found device")
  }
}
