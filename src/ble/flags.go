package ble

import (
  "strconv"
  "strings"
)

type Flags int

const (
  // Run active scans rather than passive scans (requiring explicit responses from peripherals).
  FlagScanTypeActive Flags = 1 << iota
  // Enable an allowlist for scans. Must be configured with `SetAllowListedAddresses()`.
  FlagEnableDeviceAllowList
  // Persist BLE connections in a connection pool.
  FlagPersistConnections
)

func (f Flags) String() string {
  var flags []string

  if f & FlagScanTypeActive == FlagScanTypeActive {
    flags = append(flags, "active scan")
  }

  if f & FlagEnableDeviceAllowList == FlagEnableDeviceAllowList {
    flags = append(flags, "device allow-list")
  }

  if len(flags) == 0 {
    return "none"
  }

  return strings.Join(flags, ", ")
}

type scanType uint8

const (
  scanTypePassive scanType = iota
  scanTypeActive
)

func (s scanType) String() string {
  switch s {
  case scanTypeActive:
    return "Active"
  case scanTypePassive:
    return "Passive"
  default:
    panic("unknown scanType value: " + strconv.Itoa(int(s)))
  }
}

type filterPolicy uint8

const (
  filterPolicyAcceptAll filterPolicy = iota
  filterPolicyAllowListedOnly
)

func (f filterPolicy) String() string {
  switch f {
  case filterPolicyAcceptAll:
    return "Accept All"
  case filterPolicyAllowListedOnly:
    return "Allow-listed Only"
  default:
    panic("unknown filterPolicy value: " + strconv.Itoa(int(f)))
  }
}
