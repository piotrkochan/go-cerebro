package auth

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-ldap/ldap/v3"
	"github.com/lmenezes/cerebro/internal/config"
)

type LDAPService struct {
	url          string
	userTemplate string
	baseDN       string
	groupSearch  *config.GroupSearch
	bindDN       string
	bindPW       string
}

func NewLDAPService(s config.AuthSettings) (*LDAPService, error) {
	if s.URL == "" || s.UserTemplate == "" || s.BaseDN == "" {
		return nil, errors.New("ldap auth requires url, user_template and base_dn")
	}
	svc := &LDAPService{
		url:          s.URL,
		userTemplate: s.UserTemplate,
		baseDN:       s.BaseDN,
		bindDN:       s.BindDN,
		bindPW:       s.BindPW,
	}
	if s.GroupSearch != nil && s.GroupSearch.Group != "" {
		gs := *s.GroupSearch
		if gs.BaseDN == "" {
			gs.BaseDN = s.BaseDN
		}
		if gs.UserAttrTemplate == "" {
			gs.UserAttrTemplate = s.UserTemplate
		}
		svc.groupSearch = &gs
	}
	return svc, nil
}

func (l *LDAPService) Authenticate(username, password string) (string, error) {
	if !l.checkUserAuth(username, password) {
		return "", ErrInvalidCredentials
	}
	if l.groupSearch != nil {
		if l.bindDN == "" || l.bindPW == "" {
			return "", errors.New("ldap group_search requires bind_dn and bind_pw")
		}
		if !l.checkGroupMembership(username) {
			return "", ErrInvalidCredentials
		}
	}
	return username, nil
}

func (l *LDAPService) dial() (*ldap.Conn, error) {
	return ldap.DialURL(l.url)
}

func (l *LDAPService) checkUserAuth(username, password string) bool {
	conn, err := l.dial()
	if err != nil {
		slog.Error("ldap dial failed", "err", err)
		return false
	}
	defer conn.Close()
	dn := fmt.Sprintf(l.userTemplate, username, l.baseDN)
	if err := conn.Bind(dn, password); err != nil {
		slog.Info("ldap user auth failed", "user", username, "err", err)
		return false
	}
	return true
}

func (l *LDAPService) checkGroupMembership(username string) bool {
	conn, err := l.dial()
	if err != nil {
		slog.Error("ldap dial failed", "err", err)
		return false
	}
	defer conn.Close()
	if err := conn.Bind(l.bindDN, l.bindPW); err != nil {
		slog.Error("ldap group bind failed", "err", err)
		return false
	}
	user := fmt.Sprintf(l.groupSearch.UserAttrTemplate, username, l.baseDN)
	filter := fmt.Sprintf("(& (%s=%s)(%s))", l.groupSearch.UserAttr, ldap.EscapeFilter(user), l.groupSearch.Group)
	req := ldap.NewSearchRequest(
		l.groupSearch.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,
		0,
		false,
		filter,
		[]string{"dn"},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		slog.Info("ldap group membership check failed", "user", username, "err", err)
		return false
	}
	return len(res.Entries) > 0
}
