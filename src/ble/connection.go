package ble

import (
	"context"
	"net"
	"sync"

	"github.com/go-ble/ble"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	successfulConnectionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inkbird_exporter_ble_successful_connections_total",
	})
	failedConnectionsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inkbird_exporter_ble_failed_connections_total",
	})
	connectionsFromPoolCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inkbird_exporter_ble_reused_connections_total",
	})
	disconnectsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inkbird_exporter_ble_disconnections_total",
	})
)

type connectionPool struct {
	mu sync.Mutex

	connections map[string]Client
}

func initConnectionPool() *connectionPool {
	return &connectionPool{
		connections: make(map[string]ble.Client),
	}
}

func (h *Handle) Connect(ctx context.Context, addr net.HardwareAddr) (Client, error) {
	if h.connPool == nil {
		c, err := ble.Dial(ctx, addr)

		if err == nil {
			successfulConnectionsCounter.Inc()
		} else {
			failedConnectionsCounter.Inc()
		}

		return c, err
	}

	addrStr := addr.String()

	h.connPool.mu.Lock()
	defer h.connPool.mu.Unlock()

	if conn := h.connPool.connections[addrStr]; conn != nil {
		connectionsFromPoolCounter.Inc()
		log.Trace().Stringer("Addr", addr).Msg("ble: reusing connection from connection pool")
		return conn, nil
	}

	conn, err := ble.Dial(ctx, addr)

	if err != nil {
		failedConnectionsCounter.Inc()
		return nil, err
	}

	successfulConnectionsCounter.Inc()

	h.connPool.connections[addrStr] = conn
	log.Debug().Stringer("Addr", addr).Msg("ble: successfully opened new connection to device")

	// spawn a watchdog removing the entry from the connection pool when the connection breaks.
	go func() {
		<-conn.Disconnected()

		disconnectsCounter.Inc()
		log.Debug().Stringer("Addr", addr).Msg("ble: connection with device closed, cleaning up")

		if h == nil {
			return
		}

		h.connPool.mu.Lock()
		defer h.connPool.mu.Unlock()

		delete(h.connPool.connections, addrStr)
	}()

	return conn, nil
}

// Clear the connection pool (if any) and close all connections.
func (h *Handle) DisconnectAll() {
	if h.connPool == nil {
		return
	}

	h.connPool.mu.Lock()
	defer h.connPool.mu.Unlock()

	for _, conn := range h.connPool.connections {
		conn.CancelConnection()
	}

	h.connPool.connections = make(map[string]ble.Client)
}
