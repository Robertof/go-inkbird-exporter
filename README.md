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

Inkbird devices require a BLE active scan or connection in order to read temperature data.

This exporter collects data periodically from all devices using a user-supplied interval.
Optionally, the collector goes to sleep if no reads are done by Prometheus after a timeout. This
helps to prevent excessive battery wear on your devices in case Prometheus becomes unhealthy.

## Usage

```
Usage of ./go-inkbird-exporter:
  -backoff duration
      Exponential backoff factor for retries (default 500ms)
  -bind string
      Where the exporter will bind to (default "localhost:9102")
  -bluetooth-device int
      Bluetooth (HCI) device ID
  -debug
      Enable debug logs
  -discover
      Discover available BLE devices and quit
  -idle-timeout duration
      Timeout after which the collector is shut down if no data is read. Defaults to 3 * CollectionInterval
  -initial-timeout duration
      Timeout for the collection done on start (per retry attempt) (default 3s)
  -inkbird key=value,key=value
      Device spec for this device in the form of key=value,key=value. Example: `addr=AA:BB:CC:DD:EE:FF, name=outside-temperature`
  -interval duration
      How frequently data collection happens (default 5m0s)
  -max-retries int
      Max number of retries (default 2)
  -timeout duration
      Timeout for the periodic collections (per retry attempt) (default 5s)
  -trace
      Enable trace logs
```

The `TRACE` and `DEBUG` environment variables can also be used to change the log level.

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
