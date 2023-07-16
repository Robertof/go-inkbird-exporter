package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/robertof/go-inkbird-exporter/collector"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/robertof/go-inkbird-exporter/device/inkbird"
)

type config struct {
  Debug, Trace bool
  BindAddress string
  DiscoverDevices bool
  BluetoothDeviceId int
  MaxRetries int
  InitialCollectionTimeout, CollectionTimeout time.Duration
  CollectionInterval, CollectionIdleTimeout time.Duration
  Backoff time.Duration
  Devices []device.Device
}

type deviceFactory func(device.DeviceSpec) (device.Device, error)

type boundDeviceList struct {
  name string
  factory deviceFactory
  list *[]device.Device
}

var deviceFactories = map[string]deviceFactory {
  "inkbird": func(ds device.DeviceSpec) (device.Device, error) {
    return inkbird.FromDeviceSpec(ds)
  },
}

func (d *boundDeviceList) String() string {
  return ""
}

func (d *boundDeviceList) Set(v string) error {
  ds := device.NewDeviceSpec(v)

  device, err := d.factory(ds)
  if err != nil {
    return fmt.Errorf("failed to create device: %w", err)
  }

  *d.list = append(*d.list, device)

  return nil
}

func ParseArgs() config {
  var cfg config

  flag.StringVar(&cfg.BindAddress,"bind", "localhost:9102", "Where the exporter will bind to")
  flag.IntVar(&cfg.BluetoothDeviceId, "bluetooth-device", 0, "Bluetooth (HCI) device ID")
  flag.BoolVar(&cfg.DiscoverDevices, "discover", false, "Discover available BLE devices and quit")
  flag.IntVar(&cfg.MaxRetries, "max-retries", collector.DefaultMaxRetries, "Max number of retries")
  flag.DurationVar(&cfg.InitialCollectionTimeout, "initial-timeout", 3 * time.Second,
    "Timeout for the collection done on start (per retry attempt)")
  flag.DurationVar(&cfg.CollectionTimeout, "timeout", collector.DefaultTimeoutPerAttempt,
    "Timeout for the periodic collections (per retry attempt)")
  flag.DurationVar(&cfg.CollectionInterval, "interval", 300 * time.Second,
    "How frequently data collection happens")
  flag.DurationVar(&cfg.CollectionIdleTimeout, "idle-timeout", -1,
    "Timeout after which the collector is shut down if no data is read. Defaults to 3 * CollectionInterval")
  flag.DurationVar(&cfg.Backoff, "backoff", collector.DefaultBackoffFactor,
    "Exponential backoff factor for retries")
  flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug logs")
  flag.BoolVar(&cfg.Trace, "trace", false, "Enable trace logs")

  for deviceName, deviceFactory := range deviceFactories {
    boundList := boundDeviceList{
      name:    deviceName,
      factory: deviceFactory,
      list:    &cfg.Devices,
    }

    flag.Var(&boundList, deviceName, "Device spec for this device in the form of `key=value,key=value`. Example: `addr=AA:BB:CC:DD:EE:FF, name=outside-temperature`")
  }

  flag.Parse()

  if cfg.CollectionIdleTimeout < 0 {
    cfg.CollectionIdleTimeout = cfg.CollectionInterval * 3
  }

  if !cfg.DiscoverDevices && len(cfg.Devices) == 0 {
    fmt.Fprintln(os.Stderr, "Error: at least one device is required!")
    flag.Usage()
    os.Exit(1)
  }

  return cfg
}
