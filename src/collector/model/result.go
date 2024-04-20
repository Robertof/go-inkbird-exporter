package model

import (
	"fmt"

	"github.com/robertof/go-inkbird-exporter/device"
)

type Result struct {
  Reading device.Reading
  Error error
}

func (c Result) String() string {
  if c.Error != nil {
    return fmt.Sprintf("result:error(%v)", c.Error)
  } else {
    return fmt.Sprintf("result:success(%v)", c.Reading)
  }
}

type DeviceResult struct {
	device.Device
	Result
}
