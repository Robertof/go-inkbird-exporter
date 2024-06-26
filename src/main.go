package main

import (
  "context"
  "fmt"
  "net"
  "net/http"
  "os"
  "os/signal"
  "syscall"
  "time"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/robertof/go-inkbird-exporter/ble"
  "github.com/robertof/go-inkbird-exporter/collector"
  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/robertof/go-inkbird-exporter/metrics"
  "github.com/robertof/go-inkbird-exporter/utils"
  "github.com/rs/zerolog"
  "github.com/rs/zerolog/log"
)

func main() {
  zerolog.DurationFieldUnit = time.Second
  zerolog.TimeFieldFormat = time.RFC3339Nano

  log.Logger = log.Output(zerolog.ConsoleWriter{
    Out: os.Stderr,
    TimeFormat: "15:04:05.000",
  })

  cfg := ParseArgs()

  if cfg.Trace || os.Getenv("TRACE") != "" {
      zerolog.SetGlobalLevel(zerolog.TraceLevel)
  } else if cfg.Debug || os.Getenv("DEBUG") != "" {
      zerolog.SetGlobalLevel(zerolog.DebugLevel)
  } else {
      zerolog.SetGlobalLevel(zerolog.InfoLevel)
  }

  if cfg.DiscoverDevices {
    doDeviceDiscovery(cfg)
    return
  }

  log.Info().
    Str("BindAddr", cfg.BindAddress).
    Array("Devices", utils.ToZeroLogArray(cfg.Devices)).
    Int("BluetoothDeviceID", cfg.BluetoothDeviceId).
    Msg("Starting with the specified configuration")

  observeSignals()

  bleHandle := initBle(cfg)
  initialReadings := collectInitialReadings(cfg, bleHandle)

  coll := collector.NewRecurring(bleHandle, cfg.Devices)
  coll.IdleTimeout = cfg.CollectionIdleTimeout
  coll.Update(initialReadings)

  registry := prometheus.NewRegistry()

  metrics.RegisterCollector(
    func() (map[device.Device]device.Reading, time.Time) {
      // no way to get the HTTP request context from the collector unfortunately :(
      return coll.WaitLatest(context.Background())
    },
    registry,
  )

  if cfg.EnableMetamonitoring {
    ble.RegisterMetrics(registry)
  }

  go coll.Start(
    ble.WrapContextWithSigHandler(context.WithCancel(context.Background())),
    cfg.CollectionInterval,
    collector.CollectionOptions{
      TimeoutPerAttempt: cfg.CollectionTimeout,
      MaxRetries: cfg.MaxRetries,
      BackoffFactor: cfg.Backoff,
    },
  )

  log.Info().
      Str("ListenAddress", cfg.BindAddress).
      Msg("Starting Prometheus server")

  http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

  if err := http.ListenAndServe(cfg.BindAddress, nil); err != nil {
      log.Fatal().Err(err).Msg("Unable to bind on requested address")
  }
}

func initBle(cfg config) *ble.Handle {
  var bleFlags ble.Flags = ble.FlagEnableDeviceAllowList
  deviceAddresses := make([]net.HardwareAddr, len(cfg.Devices))

  for i, dev := range cfg.Devices {
    deviceAddresses[i] = dev.Addr()

    if backend, ok := dev.Backend().(device.PassiveBackend); ok && backend.ScanType() == device.PassiveBackendScanTypeActive {
      bleFlags |= ble.FlagScanTypeActive
    }
  }

  if cfg.PersistConnections {
    bleFlags |= ble.FlagPersistConnections
  }

  bleHandle, err := ble.InitWithConnParams(cfg.BluetoothDeviceId, cfg.BluetoothConnParams, bleFlags)

  if err != nil {
    log.Fatal().Err(err).Msg("Failed to initialize Bluetooth device")
  }

  err = bleHandle.SetAllowListedAddresses(deviceAddresses)

  if err != nil {
    log.Error().Err(err).Msg("Failed to set device allow list")

    // this essentially makes the program useless, since we have the flag to enable the allow
    // list set. try to recover by re-initializing the BLE handle without that flag.
    bleHandle.Stop()
    bleFlags &= ^ble.FlagEnableDeviceAllowList

    bleHandle, err = ble.InitWithConnParams(
      cfg.BluetoothDeviceId,
      cfg.BluetoothConnParams,
      bleFlags,
    )

    if err != nil {
      log.Fatal().Err(err).Msg("Failed to re-initialize Bluetooth device")
    }
  }

  return bleHandle
}

func observeSignals() {
  c := make(chan os.Signal, 1)

  go func() {
    for s := range c {
      var nextLevel zerolog.Level

      switch s {
      case syscall.SIGUSR1:
        nextLevel = zerolog.GlobalLevel() - 1
      case syscall.SIGUSR2:
        nextLevel = zerolog.GlobalLevel() + 1
      }

      if nextLevel < zerolog.TraceLevel {
        nextLevel = zerolog.TraceLevel
      } else if nextLevel > zerolog.PanicLevel {
        nextLevel = zerolog.PanicLevel
      }

      zerolog.SetGlobalLevel(nextLevel)

      log.Warn().Stringer("Level", nextLevel).Msg("Received signal: updating log level")
    }
  }()

  signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
}

func collectInitialReadings(cfg config, bleHandle *ble.Handle) (res map[device.Device]device.Reading) {
  log.Info().
    Dur("TimeoutSec", cfg.InitialCollectionTimeout).
    Msg("Running initial collection for the provided devices")

  readings, err := collector.CollectReadingsWithOptions(
    bleHandle,
    ble.WrapContextWithSigHandler(context.WithCancel(context.Background())),
    cfg.Devices,
    collector.CollectionOptions{
      TimeoutPerAttempt: cfg.InitialCollectionTimeout,
      MaxRetries: cfg.MaxRetries,
      BackoffFactor: cfg.Backoff,
    },
  )

  if err != nil {
    log.Fatal().
      Err(err).
      Str("Readings", fmt.Sprintf("%v", readings)).
      Msg("Failed to collect initial readings")
  }

  hasError := false
  res = make(map[device.Device]device.Reading)

  for device, result := range readings {
    if result.Error != nil {
      hasError = true

      log.Error().
        Stringer("Device", device).
        Err(result.Error).
        Msg("Failed to collect reading for device")
    } else {
      log.Info().
        Stringer("Device", device).
        Stringer("Reading", result.Reading).
        Msg("Successfully collected reading for device")

      res[device] = result.Reading
    }
  }

  if hasError {
    log.Fatal().Msg("Reading for at least one device failed, refusing to start")
  }

  return res
}
