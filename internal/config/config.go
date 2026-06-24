package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

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
	Name             string   `yaml:"name"`
	Host             string   `yaml:"host"`
	Auth             *ESAuth  `yaml:"auth,omitempty"`
	HeadersWhitelist []string `yaml:"headers_whitelist,omitempty"`
}

type GroupSearch struct {
	BaseDN           string `yaml:"base_dn"`
	UserAttr         string `yaml:"user_attr"`
	UserAttrTemplate string `yaml:"user_attr_template"`
	Group            string `yaml:"group"`
}

type AuthSettings struct {
	Username     string       `yaml:"username"`
	Password     string       `yaml:"password"`
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

type Auth struct {
	Type     string       `yaml:"type"`
	Settings AuthSettings `yaml:"settings"`
}

type Server struct {
	Port            int    `yaml:"port"`
	BasePath        string `yaml:"base_path"`
	Secret          string `yaml:"secret"`
	CookieSecure    bool   `yaml:"cookie_secure"`
	MaxRequestBytes int64  `yaml:"max_request_bytes"`
}

type ES struct {
	Gzip             bool   `yaml:"gzip"`
	AllowAdHocHosts  bool   `yaml:"allow_ad_hoc_hosts"`
	MaxResponseBytes int64  `yaml:"max_response_bytes"`
	CACertFile       string `yaml:"ca_cert_file"`
	ClientCertFile   string `yaml:"client_cert_file"`
	ClientKeyFile    string `yaml:"client_key_file"`
}

type Rest struct {
	HistorySize int `yaml:"history_size"`
}

type Features struct {
	DataExplorer bool `yaml:"data_explorer"`
}

type Data struct {
	Path string `yaml:"path"`
}

type Config struct {
	Hosts    []Host   `yaml:"hosts"`
	Auth     Auth     `yaml:"auth"`
	Server   Server   `yaml:"server"`
	ES       ES       `yaml:"es"`
	Rest     Rest     `yaml:"rest"`
	Features Features `yaml:"features"`
	Data     Data     `yaml:"data"`
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
		Server: Server{Port: 9000, BasePath: "/", Secret: "change-me", CookieSecure: true, MaxRequestBytes: DefaultMaxRequestBytes},
		ES:     ES{Gzip: true, MaxResponseBytes: DefaultMaxResponseBytes},
		Rest:   Rest{HistorySize: 50},
		Data:   Data{Path: "./cerebro.db"},
		Auth:   Auth{Type: "disabled"},
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
	if v := os.Getenv("AUTH_TYPE"); v != "" {
		c.Auth.Type = v
	}
}

func (c *Config) normalize() {
	if c.Auth.Type == "" {
		c.Auth.Type = "disabled"
	}
	if c.Server.BasePath == "" {
		c.Server.BasePath = "/"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 9000
	}
	if c.Server.MaxRequestBytes <= 0 {
		c.Server.MaxRequestBytes = DefaultMaxRequestBytes
	}
	if c.ES.MaxResponseBytes <= 0 {
		c.ES.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if c.Rest.HistorySize == 0 {
		c.Rest.HistorySize = 50
	}
	if c.Data.Path == "" {
		c.Data.Path = "./cerebro.db"
	}
	for i := range c.Hosts {
		if c.Hosts[i].Name == "" {
			c.Hosts[i].Name = c.Hosts[i].Host
		}
	}
}

func (c *Config) validate() error {
	if (c.ES.ClientCertFile == "") != (c.ES.ClientKeyFile == "") {
		return fmt.Errorf("es.client_cert_file and es.client_key_file must be configured together")
	}
	for _, h := range c.Hosts {
		if err := validateHostURL(h.Host); err != nil {
			return fmt.Errorf("invalid host %q: %w", h.Name, err)
		}
	}
	switch c.Auth.Type {
	case "disabled", "none":
		return nil
	case "basic", "ldap":
		if isDefaultSecret(c.Server.Secret) {
			return fmt.Errorf("server.secret must be set to a strong non-default value when auth.type is %q", c.Auth.Type)
		}
		return nil
	default:
		return fmt.Errorf("unknown auth type: %s", c.Auth.Type)
	}
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

func (c *Config) HostNames() []string {
	names := make([]string, 0, len(c.Hosts))
	for _, h := range c.Hosts {
		names = append(names, h.Name)
	}
	return names
}
