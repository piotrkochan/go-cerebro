package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	DefaultMaxResponseBytes = int64(25 << 20)
	DefaultMaxRequestBytes  = int64(5 << 20)
)

type ESAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Host struct {
	ID               string   `yaml:"id,omitempty"`
	Name             string   `yaml:"name"`
	Host             string   `yaml:"host"`
	Auth             *ESAuth  `yaml:"auth,omitempty"`
	HeadersWhitelist []string `yaml:"headers_whitelist,omitempty"`
}

type HostRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type GroupSearch struct {
	BaseDN           string `yaml:"base_dn"`
	UserAttr         string `yaml:"user_attr"`
	UserAttrTemplate string `yaml:"user_attr_template"`
	Group            string `yaml:"group"`
	NameAttr         string `yaml:"name_attr"`
}

type BasicAuth struct {
	Enabled  bool     `yaml:"enabled"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Groups   []string `yaml:"groups,omitempty"`
}

type LDAPAuth struct {
	Enabled      bool         `yaml:"enabled"`
	URL          string       `yaml:"url"`
	CACertFile   string       `yaml:"ca_cert_file"`
	BaseDN       string       `yaml:"base_dn"`
	Method       string       `yaml:"method"`
	UserTemplate string       `yaml:"user_template"`
	BindDN       string       `yaml:"bind_dn"`
	BindPW       string       `yaml:"bind_pw"`
	InsecureLDAP bool         `yaml:"insecure_ldap"`
	GroupSearch  *GroupSearch `yaml:"group_search,omitempty"`
}

type ProxyAuth struct {
	Enabled        bool     `yaml:"enabled"`
	UserHeader     string   `yaml:"user_header"`
	GroupsHeader   string   `yaml:"groups_header"`
	GroupSeparator string   `yaml:"group_separator"`
	TrustedProxies []string `yaml:"trusted_proxies"`
}

type Auth struct {
	Basic BasicAuth `yaml:"basic,omitempty"`
	LDAP  LDAPAuth  `yaml:"ldap,omitempty"`
	Proxy ProxyAuth `yaml:"proxy,omitempty"`
}

type Server struct {
	Port                  int    `yaml:"port"`
	BasePath              string `yaml:"base_path"`
	Secret                string `yaml:"secret"`
	CookieSecure          bool   `yaml:"cookie_secure"`
	CSRFEnabled           bool   `yaml:"csrf_enabled"`
	MaxRequestBytes       int64  `yaml:"max_request_bytes"`
	TLSCertFile           string `yaml:"tls_cert_file"`
	TLSKeyFile            string `yaml:"tls_key_file"`
	HSTSEnabled           bool   `yaml:"hsts_enabled"`
	HSTSMaxAgeSeconds     int    `yaml:"hsts_max_age_seconds"`
	HSTSIncludeSubDomains bool   `yaml:"hsts_include_subdomains"`
}

type ES struct {
	Gzip             bool   `yaml:"gzip"`
	AllowAdHocHosts  bool   `yaml:"allow_ad_hoc_hosts"`
	MaxResponseBytes int64  `yaml:"max_response_bytes"`
	CACertFile       string `yaml:"ca_cert_file"`
	ClientCertFile   string `yaml:"client_cert_file"`
	ClientKeyFile    string `yaml:"client_key_file"`
	AWS              AWS    `yaml:"aws"`
}

type AWS struct {
	Enabled         bool   `yaml:"enabled"`
	Region          string `yaml:"region"`
	Service         string `yaml:"service"`
	Profile         string `yaml:"profile"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken    string `yaml:"session_token"`
}

type Rest struct {
	HistorySize int `yaml:"history_size"`
}

type Features struct {
	DataExplorer bool `yaml:"data_explorer"`
}

type RBACPolicy struct {
	Subject  string `yaml:"subject"`
	Resource string `yaml:"resource"`
	Action   string `yaml:"action"`
	Object   string `yaml:"object"`
	Effect   string `yaml:"effect"`
}

type RBACBinding struct {
	Subject string `yaml:"subject"`
	Role    string `yaml:"role"`
}

type RBAC struct {
	Enabled     bool          `yaml:"enabled"`
	DefaultRole string        `yaml:"default_role"`
	Policies    []RBACPolicy  `yaml:"policies"`
	Bindings    []RBACBinding `yaml:"bindings"`
}

type Data struct {
	Path string `yaml:"path"`
}

type Logging struct {
	Level             string `yaml:"level"`
	Format            string `yaml:"format"`
	RequestLogEnabled bool   `yaml:"request_log_enabled"`
}

type Config struct {
	Hosts    []Host   `yaml:"hosts"`
	Auth     Auth     `yaml:"auth"`
	Server   Server   `yaml:"server"`
	ES       ES       `yaml:"es"`
	Rest     Rest     `yaml:"rest"`
	Features Features `yaml:"features"`
	RBAC     RBAC     `yaml:"rbac"`
	Data     Data     `yaml:"data"`
	Logging  Logging  `yaml:"logging"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- config path is local operator input, not request-controlled data.
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	expanded := os.ExpandEnv(string(raw))

	cfg := defaults()
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	cfg.applyEnvOverrides()
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Server: Server{
			Port:                  9000,
			BasePath:              "/",
			Secret:                "change-me",
			CookieSecure:          true,
			CSRFEnabled:           true,
			MaxRequestBytes:       DefaultMaxRequestBytes,
			HSTSEnabled:           true,
			HSTSMaxAgeSeconds:     31536000,
			HSTSIncludeSubDomains: true,
		},
		ES:   ES{Gzip: true, MaxResponseBytes: DefaultMaxResponseBytes},
		Rest: Rest{HistorySize: 50},
		Data: Data{Path: "./cerebro.db"},
		Logging: Logging{
			Level:             "info",
			Format:            "text",
			RequestLogEnabled: true,
		},
	}
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("CEREBRO_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Server.Port = n
		}
	}
	if v := os.Getenv("APPLICATION_SECRET"); v != "" {
		c.Server.Secret = v
	}
}

func (c *Config) normalize() {
	if c.Server.BasePath == "" {
		c.Server.BasePath = "/"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 9000
	}
	if c.Server.MaxRequestBytes <= 0 {
		c.Server.MaxRequestBytes = DefaultMaxRequestBytes
	}
	if c.Server.HSTSEnabled && c.Server.HSTSMaxAgeSeconds <= 0 {
		c.Server.HSTSMaxAgeSeconds = 31536000
	}
	if c.ES.MaxResponseBytes <= 0 {
		c.ES.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if c.ES.AWS.Enabled && c.ES.AWS.Service == "" {
		c.ES.AWS.Service = "es"
	}
	if c.Rest.HistorySize == 0 {
		c.Rest.HistorySize = 50
	}
	c.RBAC.DefaultRole = strings.TrimSpace(c.RBAC.DefaultRole)
	for i := range c.RBAC.Policies {
		c.RBAC.Policies[i].Subject = strings.TrimSpace(c.RBAC.Policies[i].Subject)
		c.RBAC.Policies[i].Resource = strings.TrimSpace(c.RBAC.Policies[i].Resource)
		c.RBAC.Policies[i].Action = strings.TrimSpace(c.RBAC.Policies[i].Action)
		c.RBAC.Policies[i].Object = strings.TrimSpace(c.RBAC.Policies[i].Object)
		c.RBAC.Policies[i].Effect = strings.ToLower(strings.TrimSpace(c.RBAC.Policies[i].Effect))
	}
	for i := range c.RBAC.Bindings {
		c.RBAC.Bindings[i].Subject = strings.TrimSpace(c.RBAC.Bindings[i].Subject)
		c.RBAC.Bindings[i].Role = strings.TrimSpace(c.RBAC.Bindings[i].Role)
	}
	if c.Data.Path == "" {
		c.Data.Path = "./cerebro.db"
	}
	c.Logging.Level = strings.ToLower(strings.TrimSpace(c.Logging.Level))
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	c.Logging.Format = strings.ToLower(strings.TrimSpace(c.Logging.Format))
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	for i := range c.Hosts {
		if c.Hosts[i].Name == "" {
			c.Hosts[i].Name = c.Hosts[i].Host
		}
	}
}

func (c *Config) validate() error {
	if (c.Server.TLSCertFile == "") != (c.Server.TLSKeyFile == "") {
		return fmt.Errorf("server.tls_cert_file and server.tls_key_file must be configured together")
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be one of debug, info, warn or error")
	}
	switch c.Logging.Format {
	case "text", "json":
	default:
		return fmt.Errorf("logging.format must be text or json")
	}
	if (c.ES.ClientCertFile == "") != (c.ES.ClientKeyFile == "") {
		return fmt.Errorf("es.client_cert_file and es.client_key_file must be configured together")
	}
	if c.RBAC.Enabled {
		if len(c.RBAC.Policies) == 0 {
			return fmt.Errorf("rbac.policies must not be empty when rbac.enabled is true")
		}
		for i, p := range c.RBAC.Policies {
			if p.Subject == "" || p.Resource == "" || p.Action == "" || p.Object == "" {
				return fmt.Errorf("rbac.policies[%d] requires subject, resource, action and object", i)
			}
			if p.Effect != "allow" && p.Effect != "deny" {
				return fmt.Errorf("rbac.policies[%d].effect must be allow or deny", i)
			}
		}
		for i, b := range c.RBAC.Bindings {
			if b.Subject == "" || b.Role == "" {
				return fmt.Errorf("rbac.bindings[%d] requires subject and role", i)
			}
		}
	}
	if c.ES.AWS.Enabled {
		if strings.TrimSpace(c.ES.AWS.Region) == "" {
			return fmt.Errorf("es.aws.region is required when es.aws.enabled is true")
		}
		if (c.ES.AWS.AccessKeyID == "") != (c.ES.AWS.SecretAccessKey == "") {
			return fmt.Errorf("es.aws.access_key_id and es.aws.secret_access_key must be configured together")
		}
	}
	for _, h := range c.Hosts {
		if h.ID != "" && !validHostID(h.ID) {
			return fmt.Errorf("invalid host %q id %q: use lowercase letters, digits and single hyphens", h.Name, h.ID)
		}
		if err := validateHostURL(h.Host); err != nil {
			return fmt.Errorf("invalid host %q: %w", h.Name, err)
		}
	}
	if err := c.validateHostIDs(); err != nil {
		return err
	}
	if c.Auth.Basic.Enabled || c.Auth.LDAP.Enabled || c.Auth.Proxy.Enabled {
		if isDefaultSecret(c.Server.Secret) {
			return fmt.Errorf("server.secret must be set to a strong non-default value when authentication is enabled")
		}
	}
	if c.Auth.Basic.Enabled {
		if strings.TrimSpace(c.Auth.Basic.Username) == "" || c.Auth.Basic.Password == "" {
			return fmt.Errorf("auth.basic.username and auth.basic.password are required when auth.basic.enabled is true")
		}
	}
	if c.Auth.LDAP.Enabled {
		if strings.TrimSpace(c.Auth.LDAP.URL) == "" {
			return fmt.Errorf("auth.ldap.url is required when auth.ldap.enabled is true")
		}
		if strings.TrimSpace(c.Auth.LDAP.BaseDN) == "" {
			return fmt.Errorf("auth.ldap.base_dn is required when auth.ldap.enabled is true")
		}
		if strings.TrimSpace(c.Auth.LDAP.UserTemplate) == "" {
			return fmt.Errorf("auth.ldap.user_template is required when auth.ldap.enabled is true")
		}
	}
	if c.Auth.Proxy.Enabled {
		if strings.TrimSpace(c.Auth.Proxy.UserHeader) == "" {
			return fmt.Errorf("auth.proxy.user_header is required when auth.proxy.enabled is true")
		}
		if len(c.Auth.Proxy.TrustedProxies) == 0 {
			return fmt.Errorf("auth.proxy.trusted_proxies is required when auth.proxy.enabled is true")
		}
	}
	return nil
}

func validateHostURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}
	if u.User != nil {
		return fmt.Errorf("credentials in host URL are not allowed")
	}
	return nil
}

func isDefaultSecret(secret string) bool {
	switch strings.TrimSpace(secret) {
	case "", "change-me", "dev-secret-change-me":
		return true
	default:
		return false
	}
}

func (c *Config) HostByName(name string) (Host, bool) {
	for _, h := range c.Hosts {
		if h.Name == name || h.Host == name {
			return h, true
		}
	}
	return Host{}, false
}

func (c *Config) HostBySlug(slug string) (Host, bool) {
	for i, ref := range c.HostRefs() {
		if ref.Slug == slug {
			return c.Hosts[i], true
		}
	}
	return Host{}, false
}

func (c *Config) HostNames() []string {
	names := make([]string, 0, len(c.Hosts))
	for _, h := range c.Hosts {
		names = append(names, h.Name)
	}
	return names
}

func (c *Config) HostRefs() []HostRef {
	refs := make([]HostRef, 0, len(c.Hosts))
	seen := map[string]int{}
	for _, h := range c.Hosts {
		base := h.ID
		if base == "" {
			base = HostSlug(h.Name)
		}
		seen[base]++
		slug := base
		if seen[base] > 1 {
			slug = fmt.Sprintf("%s-%d", base, seen[base])
		}
		refs = append(refs, HostRef{Name: h.Name, Slug: slug})
	}
	return refs
}

func (c *Config) validateHostIDs() error {
	seen := map[string]string{}
	for _, h := range c.Hosts {
		if h.ID == "" {
			continue
		}
		if prev, ok := seen[h.ID]; ok {
			return fmt.Errorf("duplicate host id %q for hosts %q and %q", h.ID, prev, h.Name)
		}
		seen[h.ID] = h.Name
	}
	return nil
}

func validHostID(id string) bool {
	if id == "" || id[0] == '-' || id[len(id)-1] == '-' {
		return false
	}
	lastHyphen := false
	for _, r := range id {
		if r == '-' {
			if lastHyphen {
				return false
			}
			lastHyphen = true
			continue
		}
		lastHyphen = false
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func HostSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
			continue
		}
		dash = b.Len() > 0
	}
	if b.Len() == 0 {
		return "cluster"
	}
	return b.String()
}
