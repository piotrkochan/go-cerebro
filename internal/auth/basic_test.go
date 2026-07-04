package auth

import (
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBasicServiceRequiresCredentials(t *testing.T) {
	tests := []struct {
		name     string
		settings config.BasicAuth
	}{
		{name: "missing username", settings: config.BasicAuth{Password: "admin123"}},
		{name: "missing password", settings: config.BasicAuth{Username: "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBasicService(tt.settings)

			assert.EqualError(t, err, "basic auth requires username and password settings")
		})
	}
}

func TestBasicServiceAuthenticate(t *testing.T) {
	service, err := NewBasicService(config.BasicAuth{
		Username: "admin",
		Password: "admin123",
		Groups:   []string{"cerebro-admins"},
	})
	require.NoError(t, err)

	identity, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, "admin", identity.Username)
	assert.Equal(t, []string{"cerebro-admins"}, identity.Groups)
}

func TestBasicServiceAuthenticateRejectsInvalidCredentials(t *testing.T) {
	service, err := NewBasicService(config.BasicAuth{Username: "admin", Password: "admin123"})
	require.NoError(t, err)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "wrong username", username: "root", password: "admin123"},
		{name: "wrong password", username: "admin", password: "wrong"},
		{name: "both wrong", username: "root", password: "wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := service.Authenticate(tt.username, tt.password)

			assert.ErrorIs(t, err, ErrInvalidCredentials)
			assert.Empty(t, identity.Username)
			assert.Empty(t, identity.Groups)
		})
	}
}

func TestBasicServiceReturnsGroupCopy(t *testing.T) {
	service, err := NewBasicService(config.BasicAuth{
		Username: "admin",
		Password: "admin123",
		Groups:   []string{"cerebro-admins"},
	})
	require.NoError(t, err)

	first, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	first.Groups[0] = "mutated"

	second, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, []string{"cerebro-admins"}, second.Groups)
}
