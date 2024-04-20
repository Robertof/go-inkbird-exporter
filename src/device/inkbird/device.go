package inkbird

import (
	"fmt"
	"net"

	"github.com/robertof/go-inkbird-exporter/device"
)

type Device struct {
  name string
  addr net.HardwareAddr
  backend device.Backend
}

func (d *Device) Name() string {
  return d.name
}

func (d *Device) Addr() net.HardwareAddr {
  return d.addr
}

func (d *Device) Backend() device.Backend {
  return d.backend
}

func (d *Device) String() string {
  return fmt.Sprintf("inkbird[name=%q, addr=%v]", d.name, d.addr.String())
}
