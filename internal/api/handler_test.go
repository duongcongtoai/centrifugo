package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/centrifugal/centrifugo/v3/internal/rule"

	"github.com/stretchr/testify/require"
)

func TestAPIHandler(t *testing.T) {
	n := nodeWithMemoryEngine()
	defer func() { _ = n.Shutdown(context.Background()) }()

	ruleConfig := rule.DefaultConfig
	ruleContainer := rule.NewContainer(ruleConfig)
	apiExecutor := NewExecutor(n, ruleContainer, &testSurveyCaller{}, "test")

	mux := http.NewServeMux()
	mux.Handle("/api", NewHandler(n, apiExecutor, Config{}))
	server := httptest.NewServer(mux)
	defer server.Close()

	// nil body.
	req, _ := http.NewRequest("POST", server.URL+"/api", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// empty body.
	req, _ = http.NewRequest("POST", server.URL+"/api", strings.NewReader(""))
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusBadRequest)

	// valid JSON request.
	data := `{"method":"publish","params":{"channel": "test", "data":{}}}`
	req, _ = http.NewRequest("POST", server.URL+"/api", bytes.NewBuffer([]byte(data)))
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	// request with unknown method.
	data = `{"method":"unknown","params":{"channel": "test", "data":{}}}`
	req, _ = http.NewRequest("POST", server.URL+"/api", bytes.NewBuffer([]byte(data)))
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func BenchmarkAPIHandler(b *testing.B) {
	n := nodeWithMemoryEngine()
	defer func() { _ = n.Shutdown(context.Background()) }()

	ruleConfig := rule.DefaultConfig
	ruleConfig.ClientInsecure = true
	ruleContainer := rule.NewContainer(ruleConfig)

	handler := NewHandler(n, NewExecutor(n, ruleContainer, nil, "http"), Config{})

	payload := []byte(`{"method": "publish", "params": {"channel": "index", "data": 1}}`)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request, _ := http.NewRequest(http.MethodPost, "/api", bytes.NewReader(payload))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				b.Fatalf("unexpected status code %d", recorder.Code)
			}
		}
	})
}
