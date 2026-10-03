package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
)

type BasicService struct {
	users []basicUser
}

var _ PasswordAuthenticator = (*BasicService)(nil)

const (
	basicPasswordSaltLength = 16
	basicPasswordKeyLength  = 32
)

// basicPasswordHashIterations follows OWASP's PBKDF2-HMAC-SHA-256 guidance.
var basicPasswordHashIterations = 600_000

type basicUser struct {
	username     string
	usernameHash [sha256.Size]byte
	passwordHash []byte
	passwordSalt []byte
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
		passwordSalt := make([]byte, basicPasswordSaltLength)
		if _, err := rand.Read(passwordSalt); err != nil {
			return nil, err
		}
		passwordHash, err := deriveBasicPasswordHash(user.Password, passwordSalt)
		if err != nil {
			return nil, err
		}
		users = append(users, basicUser{
			username:     username,
			usernameHash: sha256.Sum256([]byte(username)),
			passwordHash: passwordHash,
			passwordSalt: passwordSalt,
			groups:       mergeGroups(s.DefaultGroups, user.Groups),
		})
	}
	return &BasicService{users: users}, nil
}

// Authenticate checks every configured username before returning to avoid early username probing.
func (b *BasicService) Authenticate(username, password string) (Identity, error) {
	usernameHash := sha256.Sum256([]byte(username))
	var matched *basicUser
	for i := range b.users {
		user := &b.users[i]
		uOK := subtle.ConstantTimeCompare(usernameHash[:], user.usernameHash[:])
		pOK := 1
		passwordHash, err := deriveBasicPasswordHash(password, user.passwordSalt)
		if err != nil || subtle.ConstantTimeCompare(passwordHash, user.passwordHash) != 1 {
			pOK = 0
		}
		if uOK&pOK == 1 {
			matched = user
		}
	}
	if matched != nil {
		return Identity{Username: matched.username, Groups: append([]string(nil), matched.groups...), Provider: "basic"}, nil
	}
	return Identity{}, ErrInvalidCredentials
}

func deriveBasicPasswordHash(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, basicPasswordHashIterations, basicPasswordKeyLength)
}
