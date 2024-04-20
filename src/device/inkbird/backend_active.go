package inkbird

import (
  "encoding/binary"
  "fmt"

  "github.com/robertof/go-inkbird-exporter/ble"
  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/robertof/go-inkbird-exporter/utils"
  "github.com/rs/zerolog/log"
)

const (
  realtimeDataHandle = 0x24
  realtimeDataUuid = 0xfff2
)

type handleFastPathStatus uint8

const (
  handleFastPathUnknown handleFastPathStatus = iota
  handleFastPathAvailable
  handleFastPathUnavailable
)

type backendActive struct {
  fastpathDisabled bool
}

func readCharacteristic(client ble.Client, c *ble.Characteristic) (r device.Reading, outErr error) {
  data, err := client.ReadCharacteristic(c)

  if err != nil {
    return r, fmt.Errorf("failed to read characteristic '%v': %w", c, err)
  }

  if len(data) < 7 {
    return r, fmt.Errorf("%w: parsed data has insufficient length: %v", device.ErrInvalidData, data)
  }

  crc := crc16(data[0:5])
  expectedCrc := binary.LittleEndian.Uint16(data[5:])

  if crc != expectedCrc {
    return r, fmt.Errorf("%w: CRC checksum mismatch (want %v, got %v)",
      device.ErrCorruptedData, expectedCrc, crc)
  }

  temp, hum := binary.LittleEndian.Uint16(data), binary.LittleEndian.Uint16(data[2:])
  probeType := data[4]

  r.Temperatures = []float32{
    float32(temp) / 100.0,
  }
  r.ProbeType = parseTHProbeType(probeType)

  if client.Name() != "tps" {
    r.HasHumidity = true
    r.RelativeHumidity = float32(hum) / 100
  }

  return r, outErr
}

func (s backendActive) Read(c ble.Client) (r device.Reading, err error) {
  // fast path: use the pre-calculated handle and test a reading. if it doesn't work, perform
  // a full profile discovery.
  if !s.fastpathDisabled {
    char := ble.Characteristic{
      UUID: ble.UUID16(realtimeDataUuid),
      ValueHandle: realtimeDataHandle,
    }

    if res, err := readCharacteristic(c, &char); !utils.ErrorIsAnyOf(err,
        device.ErrCorruptedData, device.ErrInvalidData, ble.ErrInvalidHandle) {
      return res, err
    } else {
      // if this error is due to invalid or corrupted data, treat handle as broken and fallback
      // to slow path.
      s.fastpathDisabled = true

      log.Warn().
        Err(err).
        Msg("inkbird/backend_active: attempt to directly read characteristic handle failed, " +
            "fallbacking to slowpath")
    }
  }

  p, err := c.DiscoverProfile(false)

  if err != nil {
    return r, fmt.Errorf("cannot discover profile for device: %w", err)
  }

  found := false

  for _, svc := range p.Services {
    for _, char := range svc.Characteristics {
      if char.UUID.Equal(ble.UUID16(realtimeDataUuid)) {
        found = true

        r, err = readCharacteristic(c, char)
      }
    }
  }

  if !found {
    return r, fmt.Errorf("failed to find characteristic with UUID '%x'", realtimeDataUuid)
  }

  return r, err
}
