package device

import (
  "fmt"
  "strconv"
  "strings"
)

type ProbeType uint8

const (
  ProbeTypeUnspecified ProbeType = iota
  ProbeTypeInternal
  ProbeTypeExternal
)

func (pt ProbeType) String() string {
  switch (pt) {
  case ProbeTypeUnspecified:
    return "Unspecified"
  case ProbeTypeInternal:
    return "Internal"
  case ProbeTypeExternal:
    return "External"
  default:
    panic("Unknown probe type: " + strconv.Itoa(int(pt)))
  }
}

type Reading struct {
  RelativeHumidity float32
  Temperatures []float32
  BatteryLevel uint8
  ProbeType

  HasBatteryLevel bool
  HasHumidity bool
}

func (r Reading) String() string {
  var fields []string

  if r.HasHumidity {
    fields = append(fields, fmt.Sprintf("Humidity=%.1f%%", r.RelativeHumidity))
  }

  if r.HasBatteryLevel {
    fields = append(fields, fmt.Sprintf("Battery=%d%%", r.BatteryLevel))
  }

  return fmt.Sprintf("Reading[Temperatures=%v,ProbeType=%v,%v]",
    r.Temperatures, r.ProbeType, strings.Join(fields, ","))
}
