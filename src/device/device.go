package device

import (
  "errors"
  "net"
)

var (
  ErrInvalidData = errors.New("invalid data")
  ErrCorruptedData = errors.New("corrupted data")
)


type Device interface {
  Name() string
  Addr() net.HardwareAddr
  Backend() Backend
  String() string
}
