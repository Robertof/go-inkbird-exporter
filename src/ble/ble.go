package ble

import (
  "fmt"
  "net"

  "github.com/go-ble/ble"
  "github.com/go-ble/ble/linux"
  "github.com/go-ble/ble/linux/hci/cmd"
  "github.com/prometheus/client_golang/prometheus"
  "github.com/robertof/go-inkbird-exporter/utils"
  "github.com/rs/zerolog/log"
)

const (
  ErrInvalidHandle = ble.ErrInvalidHandle
  ErrReadNotPerm = ble.ErrReadNotPerm
)

type Advertisement = ble.Advertisement
type Characteristic = ble.Characteristic
type Client = ble.Client

type Handle struct {
  dev *linux.Device
  connPool *connectionPool
}

func UUID16(i uint16) ble.UUID {
  return ble.UUID16(i)
}

func RegisterMetrics(reg prometheus.Registerer) {
  reg.MustRegister(
    successfulConnectionsCounter,
    failedConnectionsCounter,
    connectionsFromPoolCounter,
    disconnectsCounter,
  )
}

func Init(deviceId int, flags Flags) (*Handle, error) {
  return InitWithConnParams(
    deviceId,
    ConnParamsDefault,
    flags,
  )
}

func InitWithConnParams(deviceId int, connParams ConnParams, flags Flags) (*Handle, error) {
  var scanType scanType = scanTypePassive
  var filterPolicy filterPolicy = filterPolicyAcceptAll

  if flags & FlagScanTypeActive == FlagScanTypeActive {
    scanType = scanTypeActive
  }

  if flags & FlagEnableDeviceAllowList == FlagEnableDeviceAllowList {
    filterPolicy = filterPolicyAllowListedOnly
  }

  log.Debug().
    Stringer("ScanType", scanType).
    Stringer("FilterPolicy", filterPolicy).
    Stringer("ConnParams", &connParams).
    Stringer("Flags", flags).
    Int("DeviceID", deviceId).
    Msg("Initializing Bluetooth device")

  dev, err := linux.NewDevice(
    ble.OptDeviceID(deviceId),
    ble.OptScanParams(cmd.LESetScanParameters{
      LEScanType:           uint8(scanType),     // 0x00: passive, 0x01: active
      LEScanInterval:       0x0004,              // 0x0004 - 0x4000; N * 0.625msec
      LEScanWindow:         0x0004,              // 0x0004 - 0x4000; N * 0.625msec
      OwnAddressType:       0x00,                // 0x00: public, 0x01: random
      ScanningFilterPolicy: uint8(filterPolicy), // 0x00: accept all, 0x01: ignore non-allow-listed.
    }),
    ble.OptConnParams(connParams.AdapterOptions()),
  )

  if err != nil {
    return nil, fmt.Errorf("failed to init bluetooth device: %w", err)
  }

  ble.SetDefaultDevice(dev)

  h := &Handle{
    dev: dev,
  }

  if flags & FlagPersistConnections == FlagPersistConnections {
    h.connPool = initConnectionPool()
  }

  return h, nil
}

func (h *Handle) SetAllowListedAddresses(a []net.HardwareAddr) error {
  log.Debug().
    Array("DeviceAddresses", utils.ToZeroLogArray(a)).
    Msg("Allow-listing the requested Bluetooth devices")

  // clear the white list to make sure we're starting from an empty slate.
  var res cmd.LEClearWhiteListRP

  err := h.dev.HCI.Send(&cmd.LEClearWhiteList{}, &res)

  if err != nil {
    return fmt.Errorf("failed to clear allow-list: %w", err)
  }

  if res.Status != 0 {
    return fmt.Errorf("failed to clear allow-list: got status: %v", res.Status)
  }

  for _, addr := range a {
    bytes := []byte(addr)

    if len(bytes) != 6 {
      panic("got non-6 byte device MAC address!?")
    }

    var res cmd.LEAddDeviceToWhiteListRP

    err := h.dev.HCI.Send(&cmd.LEAddDeviceToWhiteList{
      AddressType: 0x00, // public
      Address:     [6]byte{
        // flip due to endianness
        bytes[5],
        bytes[4],
        bytes[3],
        bytes[2],
        bytes[1],
        bytes[0],
      },
    }, &res)

    if err != nil {
      return fmt.Errorf("failed to allow-list device %q: %w", addr.String(), err)
    }

    if res.Status != 0 {
      return fmt.Errorf("failed to allow-list device %q: got status: %v", addr.String(), res.Status)
    }
  }

  return nil
}

func (h *Handle) Stop() {
  h.dev.Stop()
}
