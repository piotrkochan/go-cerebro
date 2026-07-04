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
	groups       []string
}

func NewBasicService(s config.BasicAuth) (*BasicService, error) {
	if s.Username == "" || s.Password == "" {
		return nil, errors.New("basic auth requires username and password settings")
	}
	return &BasicService{
		usernameHash: sha256.Sum256([]byte(s.Username)),
		passwordHash: sha256.Sum256([]byte(s.Password)),
		groups:       append([]string(nil), s.Groups...),
	}, nil
}

// Authenticate hashes both fields before constant-time comparison so input length does not
// affect comparison timing.
func (b *BasicService) Authenticate(username, password string) (Identity, error) {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	uOK := subtle.ConstantTimeCompare(usernameHash[:], b.usernameHash[:]) == 1
	pOK := subtle.ConstantTimeCompare(passwordHash[:], b.passwordHash[:]) == 1
	if uOK && pOK {
		return Identity{Username: username, Groups: append([]string(nil), b.groups...), Provider: "basic"}, nil
	}
	return Identity{}, ErrInvalidCredentials
}
