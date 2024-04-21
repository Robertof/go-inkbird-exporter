package inkbird

import (
  "github.com/pkg/errors"
  "github.com/robertof/go-inkbird-exporter/ble"
  "github.com/robertof/go-inkbird-exporter/device"
)

type backendPassive struct {}

func (c backendPassive) ScanType() device.PassiveBackendScanType {
  return device.PassiveBackendScanTypeActive
}

func (c backendPassive) ParseAdvertisement(a ble.Advertisement) (reading device.Reading, err error) {
  manufacturerData := a.ManufacturerData()

  if manufacturerData == nil || len(manufacturerData) == 0 {
    return reading, device.ErrInvalidData
  }

  if len(manufacturerData) == 9 {
    return parseTHAdvertisement(a.LocalName(), manufacturerData)
  } else if isBBQDevice(a.LocalName()) {
    return parseBBQAdvertisement(a.Addr(), manufacturerData)
  } else {
    return reading, errors.Wrap(device.ErrInvalidData, "inkbird: unexpected manufacturer data")
  }
}
