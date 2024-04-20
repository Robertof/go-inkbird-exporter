# `go-inkbird-exporter`

Prometheus exporter for Inkbird temperature probe devices.
No frills, highly configurable and lightweight.

## Requirements

- Go
- Linux (doesn't work on macOS, sorry!)
- A BLE-capable adapter connected (works fine with built-in adapters on RPis)

## Quick start

Clone this and build with:

```
go build -ldflags="-s -w" -o go-inkbird-reader src/
```

To run with a single device, run:

```
./go-inkbird-reader -inkbird 'addr=AA:BB:CC:DD:EE:FF, name=ambient-temp-sensor'
```

If you don't know your device's address, run:

```
./go-inkbird-reader -discover
```

... and look for a device named "sps", "tps" or "\*BBQ".

## Details

Data from Inkbird devices can either be read through a BLE active scan or through a connection.

This exporter collects data periodically from all devices using a user-supplied interval.
Optionally, the collector goes to sleep if no reads are done by Prometheus after a timeout. This
helps to prevent excessive battery wear on your devices in case Prometheus becomes unhealthy.

By default, the exporter gets the data via a BLE active scan. If this does not work reliably,
try to switch to persistent connections via `-inkbird 'addr=..., name=..., connect=true'`. (Note
that connection-mode has only been tested for Inkbird TH2 devices. Battery measurements are not
available.)

In connection mode, passing `-bluetooth-connection-params power-saving` will use aggressive
BLE connection parameters to try and reduce battery usage for persistent connections.

## Usage

```
Usage of ./go-inkbird-exporter:
  -backoff duration
      Exponential backoff factor for retries (default 500ms)
  -bind string
      Where the exporter will bind to (default "localhost:9102")
  -bluetooth-connection-params value
      Bluetooth connection parameters (one of 'default' or 'power-saving') (default default)
  -bluetooth-device int
      Bluetooth (HCI) device ID
  -debug
      Enable debug logs
  -discover
      Discover available BLE devices and quit
  -idle-timeout duration
      Timeout after which the collector is shut down if no data is read. Defaults to 3 * CollectionInterval (default -1ns)
  -initial-timeout duration
      Timeout for the collection done on start (per retry attempt) (default 3s)
  -inkbird key=value,key=value
      Device spec for this device in the form of key=value,key=value.
      Supported parameters:
      addr (string, required): MAC address of this Inkbird device
      name (string, required): Name of this Inkbird device
      connect (bool): Connect to the device instead of scanning. Disables battery measurements. Increases reliability when battery is low.
  -interval duration
      How frequently data collection happens (default 5m0s)
  -max-retries int
      Max number of retries (default 2)
  -metamonitoring
      Enable metamonitoring metrics (default true)
  -persist-connections
      Persist Bluetooth connections between collections (default true)
  -timeout duration
      Timeout for the periodic collections (per retry attempt) (default 5s)
  -trace
      Enable trace logs
```

The `TRACE` and `DEBUG` environment variables can also be used to change the log level. In addition,
sending `SIGUSR1` to a running instance of the exporter will increase the logging level, whilst
sending `SIGUSR2` will decrease it.

## Metrics

```sh
# HELP sensor_battery_ratio Battery percentage reported by the sensor.
# TYPE sensor_battery_ratio gauge
sensor_battery_ratio{name="<device-name>"}
# HELP sensor_humidity_ratio Relative humidity reported by the sensor.
# TYPE sensor_humidity_ratio gauge
sensor_humidity_ratio{name="<device-name>"}
# HELP sensor_probe_type_info Probe type reported by the sensor. 0 = unspecified, 1 = internal, 2 = external.
# TYPE sensor_probe_type_info gauge
sensor_probe_type_info{name="<device-name>"}
# HELP sensor_temperature_celsius Temperature reported by the sensor in Celsius.
# TYPE sensor_temperature_celsius gauge
sensor_temperature_celsius{name="<device-name>",probe="<probe-num>"}
```

If the `-metamonitoring` flag is enabled (default), those additional metrics are also exported:

```sh
# HELP inkbird_exporter_ble_disconnections_total Total number of BLE disconnections.
# TYPE inkbird_exporter_ble_disconnections_total counter
inkbird_exporter_ble_disconnections_total
# HELP inkbird_exporter_ble_failed_connections_total Total number of failed BLE connections.
# TYPE inkbird_exporter_ble_failed_connections_total counter
inkbird_exporter_ble_failed_connections_total
# HELP inkbird_exporter_ble_reused_connections_total Total number of reused BLE connections.
# TYPE inkbird_exporter_ble_reused_connections_total counter
inkbird_exporter_ble_reused_connections_total
# HELP inkbird_exporter_ble_successful_connections_total Total number of successful BLE connections.
# TYPE inkbird_exporter_ble_successful_connections_total counter
inkbird_exporter_ble_successful_connections_total
```
