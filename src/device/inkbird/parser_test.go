package inkbird_test

import (
  "reflect"
  "testing"

  ble_mod "github.com/go-ble/ble"
  "github.com/robertof/go-inkbird-exporter/device"
  "github.com/robertof/go-inkbird-exporter/device/inkbird"
)

func TestTHAdvertisement_WithHumidity(t *testing.T) {
  manufacturerData := []byte{
    0xd2, 0x09, 0x61, 0x15, 0x00,
    0xc0, 0xc0, 0x64, 0x08,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 54.73,
    Temperatures:     []float32{25.14},
    BatteryLevel:     100,
    ProbeType:        device.ProbeTypeInternal,
    HasBatteryLevel:  true,
    HasHumidity:      true,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}

func TestTHAdvertisement_WithoutHumidity(t *testing.T) {
  manufacturerData := []byte{
    0xd2, 0x09, 0x61, 0x15, 0x00,
    0xc0, 0xc0, 0x64, 0x08,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
    name: "tps",
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 0,
    Temperatures:     []float32{25.14},
    BatteryLevel:     100,
    ProbeType:        device.ProbeTypeInternal,
    HasBatteryLevel:  true,
    HasHumidity:      false,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}

// thanks to https://github.com/custom-components/ble_monitor for BBQ test cases
func TestBBQAdvertisement_OneProbe(t *testing.T) {
  manufacturerData := []byte{
    0x00, 0x00, 0x00, 0x00, 0x28, 0xec, 0x9a, 0x2e, 0x65, 0xd7, 0xf0, 0x00,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
    name: "iBBQ-1",
    addr: ble_mod.NewAddr("28:EC:9A:2E:65:D7"),
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 0,
    Temperatures:     []float32{24.0},
    BatteryLevel:     0,
    ProbeType:        device.ProbeTypeExternal,
    HasBatteryLevel:  false,
    HasHumidity:      false,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}

func TestBBQAdvertisement_TwoProbes(t *testing.T) {
  manufacturerData := []byte{
    0x00, 0x00, 0x00, 0x00, 0x34, 0x14, 0xb5,
    0xab, 0xf4, 0x7b, 0xc8, 0x00, 0xd2, 0x00,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
    name: "iBBQ-2",
    addr: ble_mod.NewAddr("34:14:B5:AB:F4:7B"),
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 0,
    Temperatures:     []float32{20, 21},
    BatteryLevel:     0,
    ProbeType:        device.ProbeTypeExternal,
    HasBatteryLevel:  false,
    HasHumidity:      false,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}

func TestBBQAdvertisement_FourProbes(t *testing.T) {
  manufacturerData := []byte{
    0x00, 0x00, 0x00, 0x00, 0xa8, 0xe2, 0xc1, 0x71, 0x67,
    0x1e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
    name: "iBBQ-4",
    addr: ble_mod.NewAddr("A8:E2:C1:71:67:1E"),
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 0,
    Temperatures:     []float32{0, 0, 0, 0},
    BatteryLevel:     0,
    ProbeType:        device.ProbeTypeExternal,
    HasBatteryLevel:  false,
    HasHumidity:      false,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}

func TestBBQAdvertisement_SixProbes(t *testing.T) {
  manufacturerData := []byte{
    0x00, 0x00, 0x00, 0x00, 0x18, 0x93, 0xd7, 0x35, 0x35, 0x59, 0xd2,
    0x00, 0xf6, 0xff, 0xf6, 0xff, 0xf6, 0xff, 0xf6, 0xff, 0xf6, 0xff,
  }

  advertisement := FakeAdvertisement{
    manufacturerData: manufacturerData,
    name: "iBBQ-6",
    addr: ble_mod.NewAddr("18:93:D7:35:35:59"),
  }

  dev := inkbird.Device{}
  got, err := dev.ParseAdvertisement(advertisement)

  if err != nil {
    t.Fatalf("ParseAdvertisement(%q) got error: %v", manufacturerData, err)
  }

  want := device.Reading{
    RelativeHumidity: 0,
    Temperatures:     []float32{21, 0, 0, 0, 0, 0},
    BatteryLevel:     0,
    ProbeType:        device.ProbeTypeExternal,
    HasBatteryLevel:  false,
    HasHumidity:      false,
  }

  if !reflect.DeepEqual(got, want) {
    t.Fatalf("ParseAdvertisement(%q): got %+#v, wanted %+#v", manufacturerData, got, want)
  }
}


type FakeAdvertisement struct {
  name string
  manufacturerData []byte
  addr ble_mod.Addr
}

func (f FakeAdvertisement) LocalName() string {
  return f.name
}

func (f FakeAdvertisement) ManufacturerData() []byte {
  return f.manufacturerData
}

func (f FakeAdvertisement) ServiceData() []ble_mod.ServiceData {
  return nil
}

func (f FakeAdvertisement) Services() []ble_mod.UUID {
  return nil
}

func (f FakeAdvertisement) OverflowService() []ble_mod.UUID {
  return nil
}

func (f FakeAdvertisement) TxPowerLevel() int {
  return 0
}

func (f FakeAdvertisement) Connectable() bool {
  return false
}

func (f FakeAdvertisement) SolicitedService() []ble_mod.UUID {
  return nil
}

func (f FakeAdvertisement) RSSI() int {
  return 0
}

func (f FakeAdvertisement) Addr() ble_mod.Addr {
  return f.addr
}
