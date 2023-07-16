package inkbird

import (
  "fmt"
  "net"
  "strings"

  "github.com/robertof/go-inkbird-exporter/device"
)

type Device struct {
  name string
  addr net.HardwareAddr
}

func FromDeviceSpec(spec device.DeviceSpec) (*Device, error) {
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

  return &d, nil
}

func (d *Device) Name() string {
  return d.name
}

func (d *Device) Addr() net.HardwareAddr {
  return d.addr
}

func (d *Device) Flags() device.Flags {
  return device.FlagRequiresBleActiveScan
}

func (d *Device) String() string {
  return fmt.Sprintf("inkbird[name=%q, addr=%v]", d.name, d.addr.String())
}
