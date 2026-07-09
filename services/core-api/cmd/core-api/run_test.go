package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	return addr
}

func TestRunServesAndShutsDownGracefullyOnContextCancel(t *testing.T) {
	addr := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx, config{addr: addr}, slog.New(slog.DiscardHandler))
	}()

	// The server must come up and serve health checks.
	var response *http.Response
	var err error
	require.Eventually(t, func() bool {
		response, err = http.Get(fmt.Sprintf("http://%s/healthz", addr))
		return err == nil
	}, 5*time.Second, 25*time.Millisecond)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	// Cancelling the context must terminate run without error.
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not shut down within the grace period")
	}
}

func TestRunFailsFastWhenAddressIsUnusable(t *testing.T) {
	err := run(context.Background(), config{addr: "256.256.256.256:99999"}, slog.New(slog.DiscardHandler))
	require.Error(t, err)
}

func TestNewServerConfiguresTimeouts(t *testing.T) {
	server := newServer(config{addr: ":0"}, nil)

	require.Greater(t, server.ReadHeaderTimeout, time.Duration(0))
	require.Greater(t, server.ReadTimeout, time.Duration(0))
	require.Greater(t, server.WriteTimeout, time.Duration(0))
	require.Greater(t, server.IdleTimeout, time.Duration(0))
}

func TestOpenDatabaseTunesConnectionPool(t *testing.T) {
	db, closeDatabase, err := openDatabase(config{
		databaseDSN: "postgres://opentreasury:secret@localhost:5432/opentreasury",
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, closeDatabase()) }()

	require.Equal(t, dbMaxOpenConns, db.Stats().MaxOpenConnections)
}

func TestReadyzReports503WhenDatabaseIsDown(t *testing.T) {
	// A DSN pointing at a closed port: the server boots but is not ready.
	db, closeDatabase, err := openDatabase(config{
		databaseDSN: "postgres://opentreasury:secret@127.0.0.1:1/opentreasury?connect_timeout=1",
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, closeDatabase()) }()

	server := newServer(config{addr: ":0"}, db)

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
