package auth

import (
	"os"
	"testing"

	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	basicBcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}

func TestNewBasicServiceRequiresCredentials(t *testing.T) {
	tests := []struct {
		name     string
		settings config.BasicAuth
		wantErr  string
	}{
		{name: "missing users", settings: config.BasicAuth{}, wantErr: "basic auth requires at least one user"},
		{name: "missing username", settings: config.BasicAuth{Users: []config.BasicAuthUser{{Password: "admin123"}}}, wantErr: "basic auth users require username and password settings"},
		{name: "missing password", settings: config.BasicAuth{Users: []config.BasicAuthUser{{Username: "admin"}}}, wantErr: "basic auth users require username and password settings"},
		{name: "duplicate username", settings: config.BasicAuth{Users: []config.BasicAuthUser{
			{Username: "admin", Password: "admin123"},
			{Username: "admin", Password: "other"},
		}}, wantErr: "basic auth usernames must be unique"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBasicService(tt.settings)

			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBasicServiceAuthenticate(t *testing.T) {
	service, err := NewBasicService(config.BasicAuth{
		Users: []config.BasicAuthUser{
			{Username: "admin", Password: "admin123", Groups: []string{"cerebro-admins"}},
			{Username: "viewer", Password: "viewer123", Groups: []string{"cerebro-viewers"}},
		},
	})
	require.NoError(t, err)

	identity, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, "admin", identity.Username)
	assert.Equal(t, []string{"cerebro-admins"}, identity.Groups)

	identity, err = service.Authenticate("viewer", "viewer123")
	require.NoError(t, err)
	assert.Equal(t, "viewer", identity.Username)
	assert.Equal(t, []string{"cerebro-viewers"}, identity.Groups)
}

func TestBasicServiceAuthenticateRejectsInvalidCredentials(t *testing.T) {
	service, err := NewBasicService(config.BasicAuth{Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123"}}})
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
		Users: []config.BasicAuthUser{{Username: "admin", Password: "admin123", Groups: []string{"cerebro-admins"}}},
	})
	require.NoError(t, err)

	first, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	first.Groups[0] = "mutated"

	second, err := service.Authenticate("admin", "admin123")
	require.NoError(t, err)
	assert.Equal(t, []string{"cerebro-admins"}, second.Groups)
}
