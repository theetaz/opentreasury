//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCoreAPISmoke(t *testing.T) {
	addr := freeLocalAddress(t)
	binaryPath := buildCoreAPI(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	command := exec.CommandContext(ctx, binaryPath)
	command.Env = append(os.Environ(), "OPENTREASURY_CORE_API_ADDR="+addr)
	if err := command.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		cancel()
		_ = command.Wait()
	}()

	baseURL := "http://" + addr
	waitForHealth(t, baseURL)

	response := postJSON(t, baseURL+"/v1/transactions/validate", map[string]any{
		"id":              "txn-2026-0001",
		"institutionId":   "minfin",
		"fiscalYear":      2026,
		"amountMinor":     125000,
		"currency":        "USD",
		"description":     "Road maintenance payment",
		"transactionDate": "2026-06-28",
	})
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("response.StatusCode = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func buildCoreAPI(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	serviceRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	binaryPath := filepath.Join(t.TempDir(), "core-api")
	command := exec.Command("go", "build", "-o", binaryPath, "./cmd/core-api")
	command.Dir = serviceRoot

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go build error = %v; output = %s", err, output)
	}

	return binaryPath
}

func freeLocalAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	return listener.Addr().String()
}

func waitForHealth(t *testing.T, baseURL string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(baseURL + "/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("core API did not become healthy at %s", baseURL)
}

func postJSON(t *testing.T, url string, payload map[string]any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(payload) error = %v", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s error = %v", url, err)
	}

	return response
}
