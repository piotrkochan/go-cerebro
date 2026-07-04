package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type BasicService struct {
	users []basicUser
}

type basicUser struct {
	username     string
	usernameHash [sha256.Size]byte
	passwordHash []byte
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
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		users = append(users, basicUser{
			username:     username,
			usernameHash: sha256.Sum256([]byte(username)),
			passwordHash: passwordHash,
			groups:       append([]string(nil), user.Groups...),
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
		if bcrypt.CompareHashAndPassword(user.passwordHash, []byte(password)) != nil {
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
