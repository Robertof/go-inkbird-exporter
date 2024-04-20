package inkbird

import (
  "encoding/binary"
  "net"
  "reflect"
  "strings"

  "github.com/go-ble/ble"
  "github.com/pkg/errors"
  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/robertof/go-inkbird-exporter/utils"
  "github.com/rs/zerolog/log"
)

// props to https://github.com/tobievii/inkbird for the CRC logic
func crc16(b []byte) (ret uint16) {
  ret = 0xffff

  for _, byte := range b {
    ret ^= uint16(byte)

    for i := 0; i < 8; i += 1 {
      bit := ret & 0x1
      ret >>= 1

      if bit != 0 {
        ret ^= 0xa001
      }
    }
  }

  return ret
}

func isBBQDevice(deviceName string) bool {
  deviceName = strings.ToLower(deviceName)
  return strings.Contains(deviceName, "xbbq") || strings.Contains(deviceName, "ibbq")
}

func parseTHProbeType(t byte) device.ProbeType {
  switch t {
  case 0:
    return device.ProbeTypeInternal
  case 1:
  case 3:
    return device.ProbeTypeExternal
  }

  log.Warn().Int("ProbeType", int(t)).Msg("inkbird: received unknown probe type")
  return device.ProbeTypeUnspecified
}

func parseTHAdvertisement(deviceName string, data []byte) (reading device.Reading, err error) {
  bo := binary.LittleEndian
  crc := crc16(data[0:5])
  expectedCrc := bo.Uint16(data[5:])

  if crc != expectedCrc {
    return reading, errors.Wrapf(device.ErrCorruptedData, "unexpected CRC (wanted %d, got %v)",
      expectedCrc, crc)
  }

  rawTemp := int16(bo.Uint16(data)) // this is signed. Go does 2's complement when casting.
  rawHumidity := bo.Uint16(data[2:])
  probeType := data[4]
  battery := data[7]

  reading.Temperatures = []float32{
    float32(rawTemp) / 100.0,
  }

  if deviceName != "tps" {
    reading.HasHumidity = true
    reading.RelativeHumidity = float32(rawHumidity) / 100.0
  }

  reading.ProbeType = parseTHProbeType(probeType)
  reading.HasBatteryLevel = true
  reading.BatteryLevel = battery

  return reading, nil
}

func parseBBQAdvertisement(addr ble.Addr, data []byte) (reading device.Reading, err error) {
  if len(data) < 12 || len(data) % 2 != 0 {
    return reading, errors.Wrapf(device.ErrInvalidData,
      "unexpected data length (%d) for BBQ device, want >= 12", len(data))
  }

  numOfTemperatureProbes := (len(data) - 10) / 2

  if numOfTemperatureProbes > 6 {
    return reading, errors.Wrapf(device.ErrInvalidData,
      "found more than 6 temperature probes (%d), unknown device?", numOfTemperatureProbes)
  }

  // check MAC address embedded in data
  macBytes := data[4:10]
  hwAddr, err := net.ParseMAC(addr.String())

  if err != nil {
    return reading, errors.Wrapf(err,
      "tried to parse sender MAC address and failed!?")
  }

  if !reflect.DeepEqual(macBytes, []byte(hwAddr)) &&
     !reflect.DeepEqual(utils.Reverse(macBytes), []byte(hwAddr)) {
    return reading, errors.Wrapf(device.ErrCorruptedData,
      "device MAC address (%v) does not match MAC embedded into data (%x)", hwAddr, macBytes)
  }

  // parse sensor info
  tempInfo := data[10:]

  for i := 0; i < numOfTemperatureProbes * 2; i += 2 {
    rawTemp := int16(binary.LittleEndian.Uint16(tempInfo[i:]))

    var probeTemp float32

    if rawTemp > 0 {
      probeTemp = float32(rawTemp) / 10.0
    }

    reading.Temperatures = append(reading.Temperatures, probeTemp)
  }

  reading.ProbeType = device.ProbeTypeExternal

  return reading, nil
}
