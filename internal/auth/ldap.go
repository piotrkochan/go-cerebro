package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/lmenezes/cerebro/internal/config"
)

type LDAPService struct {
	urls           []string
	dialTimeout    time.Duration
	readTimeout    time.Duration
	userTemplate   string
	baseDN         string
	insecureLDAP   bool
	groupSearch    *config.GroupSearch
	bindDN         string
	bindPW         string
	requiredGroups []string
	defaultGroups  []string
	tlsConfig      *tls.Config
}

var _ PasswordAuthenticator = (*LDAPService)(nil)

const (
	defaultLDAPDialTimeout = 5 * time.Second
	defaultLDAPReadTimeout = 30 * time.Second
)

func NewLDAPService(s config.LDAPAuth) (*LDAPService, error) {
	if s.URL != "" && len(s.URLs) > 0 {
		return nil, errors.New("ldap auth url and urls are mutually exclusive")
	}
	urls := append([]string(nil), s.URLs...)
	if s.URL != "" {
		urls = []string{s.URL}
	}
	if len(urls) == 0 || s.UserTemplate == "" || s.BaseDN == "" {
		return nil, errors.New("ldap auth requires url or urls, user_template and base_dn")
	}
	if s.DialTimeoutSeconds < 0 || s.ReadTimeoutSeconds < 0 {
		return nil, errors.New("ldap auth timeouts must not be negative")
	}
	for i, address := range urls {
		if strings.TrimSpace(address) == "" {
			return nil, fmt.Errorf("invalid ldap urls[%d]: empty address", i)
		}
		ldapURL, err := url.Parse(address)
		if err != nil {
			return nil, fmt.Errorf("invalid ldap url %q: %w", address, err)
		}
		if ldapURL.Scheme != "ldaps" && !s.InsecureLDAP {
			return nil, errors.New("ldap auth requires ldaps:// unless insecure_ldap is explicitly enabled")
		}
	}
	tlsConfig, err := ldapTLSConfig(s.CACertFile)
	if err != nil {
		return nil, err
	}
	svc := &LDAPService{
		urls:           urls,
		dialTimeout:    defaultLDAPDialTimeout,
		readTimeout:    defaultLDAPReadTimeout,
		userTemplate:   s.UserTemplate,
		baseDN:         s.BaseDN,
		insecureLDAP:   s.InsecureLDAP,
		bindDN:         s.BindDN,
		bindPW:         s.BindPW,
		requiredGroups: append([]string(nil), s.RequiredGroups...),
		defaultGroups:  mergeGroups(s.DefaultGroups),
		tlsConfig:      tlsConfig,
	}
	if s.DialTimeoutSeconds > 0 {
		svc.dialTimeout = time.Duration(s.DialTimeoutSeconds) * time.Second
	}
	if s.ReadTimeoutSeconds > 0 {
		svc.readTimeout = time.Duration(s.ReadTimeoutSeconds) * time.Second
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
	if len(l.requiredGroups) > 0 && !hasAnyGroup(groups, l.requiredGroups) {
		return Identity{}, ErrInvalidCredentials
	}
	return Identity{Username: username, Groups: mergeGroups(l.defaultGroups, groups), Provider: "ldap"}, nil
}

func (l *LDAPService) dial() (*ldap.Conn, error) {
	var errs []error
	for _, address := range l.urls {
		opts := []ldap.DialOpt{ldap.DialWithDialer(&net.Dialer{Timeout: l.dialTimeout})}
		if parsed, _ := url.Parse(address); parsed.Scheme == "ldaps" {
			tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
			if l.tlsConfig != nil {
				tlsConfig = l.tlsConfig.Clone()
			}
			tlsConfig.ServerName = parsed.Hostname()
			opts = append(opts, ldap.DialWithTLSConfig(tlsConfig))
		}
		conn, err := ldap.DialURL(address, opts...)
		if err == nil {
			conn.SetTimeout(l.readTimeout)
			return conn, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", address, err))
	}
	return nil, errors.Join(errs...)
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

func hasAnyGroup(groups, required []string) bool {
	seen := map[string]bool{}
	for _, group := range groups {
		seen[group] = true
	}
	for _, group := range required {
		if seen[group] {
			return true
		}
	}
	return false
}
