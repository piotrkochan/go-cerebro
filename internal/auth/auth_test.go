package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIMiddleware_ReturnsUnauthorizedWithoutSession(t *testing.T) {
	mod, err := NewModule(&config.Config{
		Auth: config.Auth{
			Type:     "basic",
			Settings: config.AuthSettings{Username: "admin", Password: "admin123"},
		},
		Server: config.Server{BasePath: "/", Secret: "test-secret"},
	})
	require.NoError(t, err)
	handler := mod.APIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
