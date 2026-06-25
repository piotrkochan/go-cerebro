package elastic

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
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

func TestExecuteRequest_RejectsUnsupportedMethod(t *testing.T) {
	c := NewHTTPClient(nil)
	target := Server{Host: config.Host{Host: "http://example.com"}}

	_, err := c.ExecuteRequest(context.Background(), "CONNECT", "_cluster/health", nil, target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported REST method")
}

func TestExecuteRequest_NormalizesSupportedMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(nil)
	target := Server{Host: config.Host{Host: srv.URL}}

	_, err := c.ExecuteRequest(context.Background(), " post ", "_search", nil, target)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
}

func TestNewHTTPClientWithConfig_UsesCustomCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caFile, cert, 0o600))

	c, err := NewHTTPClientWithConfig(nil, config.ES{CACertFile: caFile})
	require.NoError(t, err)

	resp, err := c.ExecuteRequest(context.Background(), http.MethodGet, "_cluster/health", nil, Server{Host: config.Host{Host: srv.URL}})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Status)
}

func TestAWSSigningTransport_SignsRequest(t *testing.T) {
	next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Contains(t, req.Header.Get("Authorization"), "AWS4-HMAC-SHA256")
		assert.Contains(t, req.Header.Get("Authorization"), "Credential=test/")
		assert.NotEmpty(t, req.Header.Get("X-Amz-Date"))
		assert.Equal(t, emptySHA256, req.Header.Get("X-Amz-Content-Sha256"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	})

	transport := awsSigningTransport(next, config.AWS{
		Enabled: true,
		Region:  "us-east-1",
		Service: "es",
	}, mustAWSCredentialsProvider(t, config.AWS{AccessKeyID: "test", SecretAccessKey: "secret"}))

	req := httptest.NewRequest(http.MethodGet, "https://search.example.us-east-1.es.amazonaws.com/_cluster/health", nil)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAWSSigningTransport_RestoresRequestBody(t *testing.T) {
	next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"query":{"match_all":{}}}`, string(body))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    req,
		}, nil
	})

	transport := awsSigningTransport(next, config.AWS{
		Enabled: true,
		Region:  "us-east-1",
		Service: "es",
	}, mustAWSCredentialsProvider(t, config.AWS{AccessKeyID: "test", SecretAccessKey: "secret"}))

	req := httptest.NewRequest(http.MethodPost, "https://search.example.us-east-1.es.amazonaws.com/test/_search", strings.NewReader(`{"query":{"match_all":{}}}`))
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func mustAWSCredentialsProvider(t *testing.T, cfg config.AWS) aws.CredentialsProvider {
	t.Helper()
	provider, err := awsCredentialsProvider(context.Background(), cfg)
	require.NoError(t, err)
	return provider
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
