package ble

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/go-ble/ble"
	"github.com/rs/zerolog/log"
)

type ScanOptions struct {
  FilterAdvertisement func(Advertisement) bool
}

func WrapContextWithSigHandler(ctx context.Context, cancel func()) context.Context {
  return ble.WithSigHandler(ctx, cancel)
}

// Perform an active or passive scan and return every advertisement found.
func (h *Handle) ScanAll(ctx context.Context, onDevice func(Advertisement)) error {
  err := h.dev.Scan(ctx, true, onDevice)

  if err != nil {
    return fmt.Errorf("failed to initiate scan: %w", err)
  }

  return nil
}

// Perform an active or passive scan for the specified addresses and pass it to
// an handler that determines whether to accept it - ending scanning for that address -
// or rejecting it.
func (h *Handle) ScanAddresses(
  parentCtx context.Context,
  addresses []net.HardwareAddr,
  onAdvertisement func(Advertisement) bool,
) error {
  addrMap := make(map[string]chan Advertisement)

  ctx, cancel := context.WithCancel(parentCtx)
  done := make(chan string)

  for _, addr := range addresses {
    addrStr := strings.ToLower(addr.String())
    ch := make(chan ble.Advertisement, 10)
    addrMap[addrStr] = ch

    // spawn a goroutine for each device in order to serialize advertisements coming in.
    go func() {
      for {
        select {
        case next := <-ch:
          if next == nil {
            return
          }

          ok := onAdvertisement(next)

          if ok {
            done <- addrStr
            return
          }
        case <-ctx.Done():
          return
        }
      }
    }()
  }

  callback := func(a Advertisement) {
    addr := strings.ToLower(a.Addr().String())

    // the BLE lib could send an advertisement even after `Scan()` returns. do not waste
    // time enqueueing data if we're done.
    select {
    case <-ctx.Done():
      return
    default:
    }

    if ch, ok := addrMap[addr]; ok {
      log.Trace().
        Str("Advertisement", fmt.Sprintf("%+v", a)).
        Msg("ble: received advertisement, enqueueing")
      ch <- a
    }
  }

  // to avoid locking, spawn a separate goroutine whose job is just to cancel the main context
  // when all advertisements have been successfully processed.
  go func() {
    left := len(addresses)

    for {
      select {
      case <-done:
        left -= 1

        if left == 0 {
          cancel()
          return
        }
      case <-ctx.Done():
        return
      }
    }
  }()

  defer cancel()

  err := h.dev.Scan(ctx, false, callback)

  // swallow context.Canceled errors which are caused by our explicit cancellations.
  if errors.Is(err, context.Canceled) {
    err = nil
  }

  return err
}
