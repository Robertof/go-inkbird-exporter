package device

import (
	"errors"
	"net"

	"github.com/go-ble/ble"
)

var (
  ErrInvalidData = errors.New("invalid data")
  ErrCorruptedData = errors.New("corrupted data")
)

type Flags uint8

const (
  FlagRequiresBleActiveScan Flags = 1 << iota
)

type Device interface {
  Name() string
  Addr() net.HardwareAddr
  Flags() Flags
  ParseAdvertisement(a ble.Advertisement) (Reading, error)
  String() string
}
