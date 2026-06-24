package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type ESAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Host struct {
	Name             string  `yaml:"name"`
	Host             string  `yaml:"host"`
	Auth             *ESAuth `yaml:"auth,omitempty"`
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
	BaseDN       string       `yaml:"base_dn"`
	Method       string       `yaml:"method"`
	UserTemplate string       `yaml:"user_template"`
	BindDN       string       `yaml:"bind_dn"`
	BindPW       string       `yaml:"bind_pw"`
	GroupSearch  *GroupSearch `yaml:"group_search,omitempty"`
}

type Auth struct {
	Type     string       `yaml:"type"`
	Settings AuthSettings `yaml:"settings"`
}

type Server struct {
	Port     int    `yaml:"port"`
	BasePath string `yaml:"base_path"`
	Secret   string `yaml:"secret"`
}

type ES struct {
	Gzip bool `yaml:"gzip"`
}

type Rest struct {
	HistorySize int `yaml:"history_size"`
}

type Data struct {
	Path string `yaml:"path"`
}

type Config struct {
	Hosts  []Host `yaml:"hosts"`
	Auth   Auth   `yaml:"auth"`
	Server Server `yaml:"server"`
	ES     ES     `yaml:"es"`
	Rest   Rest   `yaml:"rest"`
	Data   Data   `yaml:"data"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
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
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Server: Server{Port: 9000, BasePath: "/", Secret: "change-me"},
		ES:     ES{Gzip: true},
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

func (c *Config) HostByName(name string) (Host, bool) {
	for _, h := range c.Hosts {
		if h.Name == name {
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
