package collector

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robertof/go-inkbird-exporter/ble"
	"github.com/robertof/go-inkbird-exporter/device"
	"github.com/rs/zerolog/log"
)

type signal uint8

const (
  signalWakeUp signal = iota
  signalCollectionFinished
)

type Recurring struct {
  // If no call to Latest() has been executed for more than IdleTimeout seconds, the
  // collector will suspend and resume automatically when Latest() is called again.
  IdleTimeout time.Duration

  readings map[device.Device]device.Reading
  collectionTime time.Time

  lastRead time.Time

  ble *ble.Handle
  devices []device.Device
  mu sync.Mutex

  // collector has been Start()ed
  started bool

  // collector is currently suspended due to inactivity
  suspended atomic.Bool

  signal chan signal
  wakeUpMu sync.Mutex
}

func NewRecurring(h *ble.Handle, devices []device.Device) *Recurring {
  return &Recurring{
    devices: devices,
    ble: h,
    lastRead: time.Now(),
    signal: make(chan signal),
  }
}

func (s *Recurring) Update(r map[device.Device]device.Reading) {
  s.mu.Lock()
  defer s.mu.Unlock()

  if r == nil {
    panic("attempted to set nil reading")
  }

  s.readings = r
  s.collectionTime = time.Now()
}


func (s *Recurring) wakeUpIfNeeded() bool {
  if s.suspended.Load() {
    s.signal <- signalWakeUp

    return true
  }

  return false
}


func (s *Recurring) wakeUpAndBlockIfNeeded(ctx context.Context) {
  // wait if another goroutine has already sent the wake up signal.
  s.wakeUpMu.Lock()
  defer s.wakeUpMu.Unlock()

  if s.wakeUpIfNeeded() {
    // wait until wake up is complete to proceed and block other goroutines trying to do
    // blocking reads.
    select {
    case <-ctx.Done():
    case sig := <-s.signal:
      if sig != signalCollectionFinished {
        panic("unexpected signal")
      }
    }
  }
}

func (s *Recurring) get() (map[device.Device]device.Reading, time.Time) {
  s.mu.Lock()
  defer s.mu.Unlock()

  if s.readings == nil || s.collectionTime.IsZero() {
    panic("Latest() on collector.Recurring called when not initialised yet")
  }

  s.lastRead = time.Now()

  // safe to return as we replace the old map with a new one on update.
  return s.readings, s.collectionTime
}

// Retrieve the latest collected value. Wakes up the collector if asleep.
// Doesn't wait for a new result if the collector is asleep and is waken up.
func (s *Recurring) Latest() (map[device.Device]device.Reading, time.Time) {
  s.wakeUpIfNeeded()

  return s.get()
}

// Retrieve the latest collected value. Wakes up the collector if asleep and
// waits until it finishes the collection, otherwise, returns the last available
// data without blocking.
func (s *Recurring) WaitLatest(ctx context.Context) (map[device.Device]device.Reading, time.Time) {
  s.wakeUpAndBlockIfNeeded(ctx)

  return s.get()
}


func (s *Recurring) shouldSuspend() (suspend bool, elapsed time.Duration) {
  if s.IdleTimeout == 0 {
    return false, 0
  }

  s.mu.Lock()
  defer s.mu.Unlock()

  elapsed = time.Now().Sub(s.lastRead)

  return elapsed > s.IdleTimeout, elapsed
}

func (s *Recurring) shutdown() {
  log.Info().Msg("Recurring collector is shutting down")

  close(s.signal)
}

func (s *Recurring) Start(
  ctx context.Context,
  interval time.Duration,
  opts CollectionOptions,
) {
  if s.started {
    panic("attempted to call collector.Recurring.Start() twice")
  }

  s.started = true

  log.Info().
    Dur("Interval", interval).
    Int("MaxRetries", opts.MaxRetries).
    Dur("TimeoutPerAttemptSec", opts.TimeoutPerAttempt).
    Dur("IdleTimeoutSec", s.IdleTimeout).
    Msg("Starting recurring collector")

  for {
    select {
    case <-ctx.Done():
      s.shutdown()
      return
    case <-time.After(interval):
    }

    wokeUp := false

    // check if no data has been read for too long and suspend if so.
    if suspend, elapsed := s.shouldSuspend(); suspend {
      if !s.suspended.CompareAndSwap(false, true) {
        panic("s.shouldSuspended() == true but we're alreadys suspended!?")
      }

      log.Warn().
        Dur("IdleTimeoutSec", s.IdleTimeout).
        Dur("TimeSinceLastReadSec", elapsed).
        Msg("Suspending recurring collector due to inactivity. If you see this message often, " +
            "you probably need to adjust the collection interval with '-interval'.")

      // disconnect all devices when idling
      s.ble.DisconnectAll()

      // wait until resumed
      select {
      case <-ctx.Done():
        s.shutdown()
        return
      case sig := <-s.signal:
        if sig != signalWakeUp {
          panic("unexpected signal")
        }

        if !s.suspended.CompareAndSwap(true, false) {
          panic("collector woke up from sleep but was not suspended!?")
        }

        wokeUp = true

        log.Trace().Msg("Collector woke up from sleep - starting immediate collection")
      }
    } else {
      log.Trace().Dur("Interval", interval).Msg("Recurring collector tick: collecting...")
    }

    collectionResult, err := CollectReadingsWithOptions(s.ble, ctx, s.devices, opts)

    if collectionResult != nil {
      update := make(map[device.Device]device.Reading)

      for dev, res := range collectionResult {
        if res.Error != nil {
          log.Warn().
            Stringer("Device", dev).
            Err(res.Error).
            Msg("Collection failed for device")
        } else {
          log.Debug().
            Stringer("Device", dev).
            Stringer("Reading", res.Reading).
            Msg("Successfully collected data from device")

          update[dev] = res.Reading
        }
      }

      if len(update) < len(s.devices) {
        log.Warn().
          Err(err).
          Msg("Collection failed for one or more devices!")
      }

      if len(update) > 0 {
        // do not call s.update() with empty data. we can keep returning stale data as needed.
        // as long as it has the correct timestamp, Prometheus should not report it as new.
        s.Update(update)
      }
    } else {
      log.Error().
        Err(err).
        Msg("Collection failed with undefined collection results - this should never happen!")
    }

    if wokeUp {
      select {
      case s.signal <- signalCollectionFinished:
      default:
      }
    }
  }
}
