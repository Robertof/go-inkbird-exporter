package device

import (
	"strings"

	"github.com/rs/zerolog/log"
)

type DeviceSpec map[string]string

const (
  DeviceSpecFieldName = "name"
  DeviceSpecFieldAddress = "addr"
)

func NewDeviceSpec(s string) DeviceSpec {
  spec := DeviceSpec{}
  entries := strings.Split(s, ",")

  for _, entry := range entries {
    parts := strings.SplitN(entry, "=", 2)

    if len(parts) != 2 {
      log.Warn().Str("Entry", entry).Msg("Skipping invalid device spec entry")
      continue
    }

    spec[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
  }

  return spec
}

func (ds DeviceSpec) Name() string {
  return ds[DeviceSpecFieldName]
}

func (ds DeviceSpec) Addr() string {
  return ds[DeviceSpecFieldAddress]
}
