package elastic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_BasicAuthAndHeaders(t *testing.T) {
	var gotAuth string
	var gotHeader string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotHeader = r.Header.Get("X-Proxy-User")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(nil)
	target := Server{
		Host: config.Host{
			Name: "t", Host: srv.URL,
			Auth:             &config.ESAuth{Username: "u", Password: "p"},
			HeadersWhitelist: []string{"X-Proxy-User"},
		},
		Headers: [][2]string{{"X-Proxy-User", "alice"}},
	}
	resp, err := c.GetAliases(context.Background(), target)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.Status)
	assert.True(t, resp.IsSuccess())
	assert.Equal(t, "/_aliases", gotPath)
	assert.NotEmpty(t, gotAuth)
	assert.Equal(t, "alice", gotHeader)
}

func TestExecuteRequest_NDJSONForString(t *testing.T) {
	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-type")
		gotBody, _ = readAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(nil)
	target := Server{Host: config.Host{Host: srv.URL}}

	bulk := json.RawMessage(`"{\"index\":{}}\n{\"foo\":\"bar\"}\n"`)
	_, err := c.ExecuteRequest(context.Background(), "POST", "_bulk", bulk, target)
	require.NoError(t, err)
	assert.Equal(t, "application/x-ndjson", gotCT)
	assert.Equal(t, "{\"index\":{}}\n{\"foo\":\"bar\"}\n", string(gotBody))
}

func readAll(rc interface {
	Read(p []byte) (int, error)
	Close() error
}) ([]byte, error) {
	defer rc.Close()
	out := make([]byte, 0, 256)
	buf := make([]byte, 256)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return out, nil
			}
			return out, err
		}
	}
}
