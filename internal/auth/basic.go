package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
)

type BasicService struct {
	users []basicUser
}

type basicUser struct {
	username     string
	usernameHash [sha256.Size]byte
	passwordHash [sha256.Size]byte
	groups       []string
}

func NewBasicService(s config.BasicAuth) (*BasicService, error) {
	if len(s.Users) == 0 {
		return nil, errors.New("basic auth requires at least one user")
	}
	users := make([]basicUser, 0, len(s.Users))
	seen := map[string]bool{}
	for _, user := range s.Users {
		username := strings.TrimSpace(user.Username)
		if username == "" || user.Password == "" {
			return nil, errors.New("basic auth users require username and password settings")
		}
		if seen[username] {
			return nil, errors.New("basic auth usernames must be unique")
		}
		seen[username] = true
		users = append(users, basicUser{
			username:     username,
			usernameHash: sha256.Sum256([]byte(username)),
			passwordHash: sha256.Sum256([]byte(user.Password)),
			groups:       append([]string(nil), user.Groups...),
		})
	}
	return &BasicService{users: users}, nil
}

// Authenticate hashes both fields before constant-time comparison so input length does not
// affect comparison timing.
func (b *BasicService) Authenticate(username, password string) (Identity, error) {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	var matched *basicUser
	for i := range b.users {
		user := &b.users[i]
		uOK := subtle.ConstantTimeCompare(usernameHash[:], user.usernameHash[:])
		pOK := subtle.ConstantTimeCompare(passwordHash[:], user.passwordHash[:])
		if uOK&pOK == 1 {
			matched = user
		}
	}
	if matched != nil {
		return Identity{Username: matched.username, Groups: append([]string(nil), matched.groups...), Provider: "basic"}, nil
	}
	return Identity{}, ErrInvalidCredentials
}
