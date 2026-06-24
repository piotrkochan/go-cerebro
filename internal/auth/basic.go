package auth

import (
	"crypto/subtle"
	"errors"

	"github.com/lmenezes/cerebro/internal/config"
)

type BasicService struct {
	username []byte
	password []byte
}

func NewBasicService(s config.AuthSettings) (*BasicService, error) {
	if s.Username == "" || s.Password == "" {
		return nil, errors.New("basic auth requires username and password settings")
	}
	return &BasicService{username: []byte(s.Username), password: []byte(s.Password)}, nil
}

// Authenticate compares both fields with constant-time comparison to avoid leaking the
// expected username/password through response timing.
func (b *BasicService) Authenticate(username, password string) (string, error) {
	uOK := subtle.ConstantTimeCompare([]byte(username), b.username) == 1
	pOK := subtle.ConstantTimeCompare([]byte(password), b.password) == 1
	if uOK && pOK {
		return username, nil
	}
	return "", ErrInvalidCredentials
}
