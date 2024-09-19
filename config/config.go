package config

import (
	"log/slog"
	"strings"

	"github.com/knadh/koanf/v2"

	toml "github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	BindAddress string `mapstructure:"bind_address"`
	Port        string `mapstructure:"listen_port"`
	BaseURL     string `mapstructure:"url_base"`
	// Deprecated
	ProxyProtocolPort       string   `mapstructure:"proxyprotocol_port"`
	EnableProxyprotocol     bool     `mapstructure:"enable_proxyprotocol"`
	ProxyprotocolAllowedIPs []string `mapstructure:"proxyprotocol_allowed_ips"`

	ServerLat    float64 `mapstructure:"server_lat"`
	ServerLng    float64 `mapstructure:"server_lng"`
	IPInfoAPIKey string  `mapstructure:"ipinfo_api_key"`

	StatsPassword string `mapstructure:"statistics_password"`
	RedactIP      bool   `mapstructure:"redact_ip_addresses"`

	AssetsPath string `mapstructure:"assets_path"`

	DatabaseType     string `mapstructure:"database_type"`
	DatabaseHostname string `mapstructure:"database_hostname"`
	DatabaseName     string `mapstructure:"database_name"`
	DatabaseUsername string `mapstructure:"database_username"`
	DatabasePassword string `mapstructure:"database_password"`

	DatabaseFile string `mapstructure:"database_file"`

	EnableHTTP2 bool   `mapstructure:"enable_http2"`
	EnableTLS   bool   `mapstructure:"enable_tls"`
	TLSCertFile string `mapstructure:"tls_cert_file"`
	TLSKeyFile  string `mapstructure:"tls_key_file"`
}

var (
	config *Config = &Config{
		Port:                    "8989",
		EnableProxyprotocol:     false,
		ProxyprotocolAllowedIPs: []string{"127.0.0.1/32", "::1/128"},
		StatsPassword:           "PASSWORD",
		DatabaseType:            "postgresql",
		DatabaseHostname:        "localhost",
		DatabaseName:            "speedtest",
		DatabaseUsername:        "postgres",
	}
)

func Load(configPath string) (*Config, error) {
	var k = koanf.New(".")

	if configPath == "" {
		if err := k.Load(file.Provider("settings.toml"), toml.Parser()); err != nil {
			slog.Info("no config found, using defaults", slog.Any("error", err))
		}
	} else {
		if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
			slog.Error("loading from config",
				slog.String("path", configPath),
				slog.Any("error", err))
			return nil, err
		}
	}

	if err := k.Load(env.Provider("SPEEDTEST_", ".", func(s string) string {
		ret := strings.TrimPrefix(s, "SPEEDTEST_")
		ret = strings.ToLower(ret)
		return ret
	}), nil); err != nil {
		slog.Error("loading from env", slog.Any("error", err))
	}

	if err := k.UnmarshalWithConf("", config, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		slog.Error("unmarshal to config", slog.Any("error", err))
	}
	return config, nil
}

func LoadedConfig() *Config {
	return config
}
