package device

import (
	"github.com/robertof/go-inkbird-exporter/ble"
)

// PassiveBackendScanType is the BLE scan type used to discover the device.
type PassiveBackendScanType uint8

const (
  PassiveBackendScanTypePassive PassiveBackendScanType = iota
  PassiveBackendScanTypeActive
)

// PassiveBackend represents a device that parses data passively -- that is, entirely using
// advertisements without establishing a connection to the device.
type PassiveBackend interface {
  ScanType() PassiveBackendScanType
  ParseAdvertisement(a ble.Advertisement) (Reading, error)
}

// ActiveBackend represents a device that parses data using an established device connection.
type ActiveBackend interface {
	Read(c ble.Client) (Reading, error)
}

type Backend any
