package ble

import (
  "fmt"
  "slices"

  "github.com/go-ble/ble/linux/hci/cmd"
)

type ConnParams string

const (
  ConnParamsDefault     ConnParams = "default"
  ConnParamsPowerSaving ConnParams = "power-saving"
)

// *flag.Value
func (c *ConnParams) String() string {
  return string(*c)
}

func (c *ConnParams) Set(v string) error {
  if v == "" {
    *c = ConnParamsDefault
    return nil
  }

  allParams := []ConnParams{ConnParamsDefault, ConnParamsPowerSaving}
  p := ConnParams(v)

  if !slices.Contains(allParams, p) {
    return fmt.Errorf("unknown connection param %v (must be one of %v)", p, allParams)
  }

  *c = p
  return nil
}

func (c ConnParams) AdapterOptions() cmd.LECreateConnection {
  p := cmd.LECreateConnection{
    LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
    LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
    InitiatorFilterPolicy: 0x00,      // White list is not used
    PeerAddressType:       0x00,      // Public Device Address
    PeerAddress:           [6]byte{}, //
    OwnAddressType:        0x00,      // Public Device Address
    ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
    ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
    ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
    SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
    MinimumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
    MaximumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
  }

  switch c {
  case ConnParamsDefault:
    break
  case ConnParamsPowerSaving:
    // https://developer.apple.com/accessories/Accessory-Design-Guidelines.pdf
    // section "Connection Parameters"
    // - supervision timeout between 6 to 18 secs
    // - interval max * (latency + 1) <= 6 secs
    // - supervision timeout > interval max * (latency + 1) * 3
    // ---
    // addendum from https://www.bluetooth.com/wp-content/uploads/Files/Specification/HTML/Core-54/out/en/low-energy-controller/link-layer-specification.html#UUID-2e99c85e-1cf9-e911-9837-8ca01d376541:
    // - interval max * (latency + 1) <= 1/2 supervision timeout
    // the parameters below achieve a good balance between slowness and speed.
    p.ConnIntervalMin    = 0x00f0 // 300ms
    p.ConnIntervalMax    = 0x00f0 // 300ms
    p.ConnLatency        = 0x0014 // 20
    p.SupervisionTimeout = 0x0708 // 18s
  default:
    panic("unknown Bluetooth connection param: " + c)
  }

  return p
}
