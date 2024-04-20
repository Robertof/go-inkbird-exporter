package metrics

import (
  "strconv"
  "time"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/robertof/go-inkbird-exporter/device"
)

var (
  descTemperature = prometheus.NewDesc(
    "sensor_temperature_celsius",
    "Temperature reported by the sensor in Celsius.",
    []string{"name", "probe"},
    nil,
  )

  descHumidity = prometheus.NewDesc(
    "sensor_humidity_ratio",
    "Relative humidity reported by the sensor.",
    []string{"name"},
    nil,
  )

  descBattery = prometheus.NewDesc(
    "sensor_battery_ratio",
    "Battery percentage reported by the sensor.",
    []string{"name"},
    nil,
  )

  descProbeType = prometheus.NewDesc(
    "sensor_probe_type_info",
    "Probe type reported by the sensor. 0 = unspecified, 1 = internal, 2 = external.",
    []string{"name"},
    nil,
  )
)

type CollectFunc func() (map[device.Device]device.Reading, time.Time)

type collector struct {
  CollectFunc
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
  prometheus.DescribeByCollect(c, ch)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
  out, ts := c.CollectFunc()

  if out == nil {
    panic("collector got empty data!")
  }

  for device, reading := range out {
    for probe, temp := range reading.Temperatures {
      temperature := prometheus.MustNewConstMetric(
        descTemperature,
        prometheus.GaugeValue,
        float64(temp),
        device.Name(),
        strconv.Itoa(probe),
      )

      ch <- prometheus.NewMetricWithTimestamp(ts, temperature)
    }

    if reading.HasHumidity {
      humidity := prometheus.MustNewConstMetric(
        descHumidity,
        prometheus.GaugeValue,
        float64(reading.RelativeHumidity) / 100,
        device.Name(),
      )

      ch <- prometheus.NewMetricWithTimestamp(ts, humidity)
    }

    if reading.HasBatteryLevel {
      battery := prometheus.MustNewConstMetric(
        descBattery,
        prometheus.GaugeValue,
        float64(reading.BatteryLevel) / 100,
        device.Name(),
      )

      ch <- prometheus.NewMetricWithTimestamp(ts, battery)
    }

    probeType := prometheus.MustNewConstMetric(
      descProbeType,
      prometheus.GaugeValue,
      float64(reading.ProbeType),
      device.Name(),
    )

    ch <- prometheus.NewMetricWithTimestamp(ts, probeType)
  }
}

func RegisterCollector(f CollectFunc, reg prometheus.Registerer) {
  c := &collector{f}

  reg.MustRegister(c)
}
