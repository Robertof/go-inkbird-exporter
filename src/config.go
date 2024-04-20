package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/robertof/go-inkbird-exporter/ble"
	"github.com/robertof/go-inkbird-exporter/collector"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/robertof/go-inkbird-exporter/device/inkbird"
)

type config struct {
  Debug, Trace bool
  BindAddress string
  EnableMetamonitoring bool
  DiscoverDevices bool
  BluetoothDeviceId int
  BluetoothConnParams ble.ConnParams
  PersistConnections bool
  MaxRetries int
  InitialCollectionTimeout, CollectionTimeout time.Duration
  CollectionInterval, CollectionIdleTimeout time.Duration
  Backoff time.Duration
  Devices []device.Device
}

type boundDeviceList struct {
  device.Factory
  name string
  list *[]device.Device
}

var deviceFactories = map[string]device.Factory {
  "inkbird": &inkbird.Factory{},
}

func (d *boundDeviceList) String() string {
  return ""
}

func (d *boundDeviceList) Set(v string) error {
  ds := device.NewDeviceSpec(v)

  device, err := d.FromSpec(ds)
  if err != nil {
    return fmt.Errorf("failed to create device: %w", err)
  }

  *d.list = append(*d.list, device)

  return nil
}

func ParseArgs() config {
  var cfg config

  cfg.BluetoothConnParams = ble.ConnParamsDefault

  flag.StringVar(&cfg.BindAddress,"bind", "localhost:9102", "Where the exporter will bind to")
  flag.IntVar(&cfg.BluetoothDeviceId, "bluetooth-device", 0, "Bluetooth (HCI) device ID")
  flag.Var(&cfg.BluetoothConnParams, "bluetooth-connection-params", "Bluetooth connection parameters (one of 'default' or 'power-saving')")
  flag.BoolVar(&cfg.PersistConnections, "persist-connections", true, "Persist Bluetooth connections between collections")
  flag.BoolVar(&cfg.DiscoverDevices, "discover", false, "Discover available BLE devices and quit")
  flag.BoolVar(&cfg.EnableMetamonitoring, "metamonitoring", true, "Enable metamonitoring metrics")
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
      Factory: deviceFactory,
      list:    &cfg.Devices,
    }

    help := "Device spec for this device in the form of `key=value,key=value`."

    if docs, ok := deviceFactory.(device.FactoryDocs); ok {
      help += "\n" + docs.Help()
    }

    flag.Var(&boundList, deviceName, help)
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
