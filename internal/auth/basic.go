package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"

	"github.com/lmenezes/cerebro/internal/config"
)

type BasicService struct {
	usernameHash [sha256.Size]byte
	passwordHash [sha256.Size]byte
}

func NewBasicService(s config.AuthSettings) (*BasicService, error) {
	if s.Username == "" || s.Password == "" {
		return nil, errors.New("basic auth requires username and password settings")
	}
	return &BasicService{
		usernameHash: sha256.Sum256([]byte(s.Username)),
		passwordHash: sha256.Sum256([]byte(s.Password)),
	}, nil
}

// Authenticate hashes both fields before constant-time comparison so input length does not
// affect comparison timing.
func (b *BasicService) Authenticate(username, password string) (string, error) {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	uOK := subtle.ConstantTimeCompare(usernameHash[:], b.usernameHash[:]) == 1
	pOK := subtle.ConstantTimeCompare(passwordHash[:], b.passwordHash[:]) == 1
	if uOK && pOK {
		return username, nil
	}
	return "", ErrInvalidCredentials
}
