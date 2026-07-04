package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/lmenezes/cerebro/internal/config"
)

type LDAPService struct {
	url          string
	userTemplate string
	baseDN       string
	insecureLDAP bool
	groupSearch  *config.GroupSearch
	bindDN       string
	bindPW       string
	tlsConfig    *tls.Config
}

func NewLDAPService(s config.LDAPAuth) (*LDAPService, error) {
	if s.URL == "" || s.UserTemplate == "" || s.BaseDN == "" {
		return nil, errors.New("ldap auth requires url, user_template and base_dn")
	}
	ldapURL, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid ldap url: %w", err)
	}
	if ldapURL.Scheme != "ldaps" && !s.InsecureLDAP {
		return nil, errors.New("ldap auth requires ldaps:// unless insecure_ldap is explicitly enabled")
	}
	tlsConfig, err := ldapTLSConfig(s.CACertFile)
	if err != nil {
		return nil, err
	}
	svc := &LDAPService{
		url:          s.URL,
		userTemplate: s.UserTemplate,
		baseDN:       s.BaseDN,
		insecureLDAP: s.InsecureLDAP,
		bindDN:       s.BindDN,
		bindPW:       s.BindPW,
		tlsConfig:    tlsConfig,
	}
	if s.GroupSearch != nil && s.GroupSearch.UserAttr != "" {
		gs := *s.GroupSearch
		if gs.BaseDN == "" {
			gs.BaseDN = s.BaseDN
		}
		if gs.UserAttrTemplate == "" {
			gs.UserAttrTemplate = s.UserTemplate
		}
		if gs.NameAttr == "" {
			gs.NameAttr = "cn"
		}
		svc.groupSearch = &gs
	}
	return svc, nil
}

func (l *LDAPService) Authenticate(username, password string) (Identity, error) {
	if !l.checkUserAuth(username, password) {
		return Identity{}, ErrInvalidCredentials
	}
	groups := []string{}
	if l.groupSearch != nil {
		if l.bindDN == "" || l.bindPW == "" {
			return Identity{}, errors.New("ldap group_search requires bind_dn and bind_pw")
		}
		var err error
		groups, err = l.discoverGroups(username)
		if err != nil {
			return Identity{}, err
		}
		if len(groups) == 0 {
			return Identity{}, ErrInvalidCredentials
		}
	}
	return Identity{Username: username, Groups: groups}, nil
}

func (l *LDAPService) dial() (*ldap.Conn, error) {
	if l.tlsConfig != nil {
		return ldap.DialURL(l.url, ldap.DialWithTLSConfig(l.tlsConfig))
	}
	return ldap.DialURL(l.url)
}

func ldapTLSConfig(caCertFile string) (*tls.Config, error) {
	if caCertFile == "" {
		return nil, nil
	}
	pem, err := os.ReadFile(caCertFile) // #nosec G304 -- operator-controlled config path.
	if err != nil {
		return nil, fmt.Errorf("read ldap ca_cert_file: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("ldap ca_cert_file does not contain a valid PEM certificate")
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, nil
}

func (l *LDAPService) checkUserAuth(username, password string) bool {
	if !safeLDAPTemplateValue(username) {
		slog.Info("ldap rejected unsafe username")
		return false
	}
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

func (l *LDAPService) discoverGroups(username string) ([]string, error) {
	if !safeLDAPTemplateValue(username) {
		slog.Info("ldap rejected unsafe username")
		return nil, ErrInvalidCredentials
	}
	conn, err := l.dial()
	if err != nil {
		slog.Error("ldap dial failed", "err", err)
		return nil, ErrInvalidCredentials
	}
	defer conn.Close()
	if err := conn.Bind(l.bindDN, l.bindPW); err != nil {
		slog.Error("ldap group bind failed", "err", err)
		return nil, ErrInvalidCredentials
	}
	user := fmt.Sprintf(l.groupSearch.UserAttrTemplate, username, l.baseDN)
	groupFilter := l.groupSearch.Group
	if groupFilter == "" {
		groupFilter = "(objectClass=*)"
	}
	filter := fmt.Sprintf("(&(%s=%s)%s)", l.groupSearch.UserAttr, ldap.EscapeFilter(user), groupFilter)
	req := ldap.NewSearchRequest(
		l.groupSearch.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{l.groupSearch.NameAttr},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		slog.Info("ldap group membership check failed", "user", username, "err", err)
		return nil, ErrInvalidCredentials
	}
	groups := make([]string, 0, len(res.Entries)*2)
	for _, entry := range res.Entries {
		if entry.DN != "" {
			groups = append(groups, entry.DN)
		}
		for _, value := range entry.GetAttributeValues(l.groupSearch.NameAttr) {
			if value != "" {
				groups = append(groups, value)
			}
		}
	}
	return groups, nil
}

func safeLDAPTemplateValue(value string) bool {
	if value == "" || strings.ContainsAny(value, ",=+<>#;\"\\\x00") {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
