package inkbird

import (
  "fmt"
  "net"
  "strings"

  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/rs/zerolog/log"
)

type Factory struct{}

func (f *Factory) FromSpec(spec device.DeviceSpec) (device.Device, error) {
  d := Device{}

  addr := spec.Addr()

  if name := spec.Name(); name != "" {
    d.name = name
  } else {
    d.name = "inkbird-" + strings.ToLower(strings.ReplaceAll(addr, ":", ""))
  }

  hwAddr, err := net.ParseMAC(addr)
  if err != nil {
    return nil, fmt.Errorf("invalid addr: %w", err)
  }

  d.addr = hwAddr

  if connect := spec["connect"]; connect == "yes" || connect == "true" {
    log.Debug().Stringer("Device", &d).Msg("inkbird: using active backend (reading w/connection)")
    d.backend = &backendActive{}
  } else {
    log.Debug().Stringer("Device", &d).Msg("inkbird: using passive backend (reading w/scan)")
    d.backend = &backendPassive{}
  }

  return &d, nil
}

func (f *Factory) Help() string {
  return `Supported parameters:
addr (string, required): MAC address of this Inkbird device
name (string, required): Name of this Inkbird device
connect (bool): Connect to the device instead of scanning. Disables battery measurements. Increases reliability when battery is low.`
}
