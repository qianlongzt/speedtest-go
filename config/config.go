package config

import (
	"flag"
	"log/slog"
	"strings"

	"github.com/itzg/go-flagsfiller"
	toml "github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	BindAddress string `flag:"bind_address"`
	Port        string `flag:"listen_port"`
	BaseURL     string `flag:"url_base"`
	// Deprecated
	ProxyProtocolPort       string   `flag:"proxyprotocol_port"`
	EnableProxyprotocol     bool     `flag:"enable_proxyprotocol"`
	ProxyprotocolAllowedIPs []string `flag:"proxyprotocol_allowed_ips"`

	ServerLat    float64 `flag:"server_lat"`
	ServerLng    float64 `flag:"server_lng"`
	IPInfoAPIKey string  `flag:"ipinfo_api_key"`

	StatsPassword string `flag:"statistics_password"`
	RedactIP      bool   `flag:"redact_ip_addresses"`

	AssetsPath string `flag:"assets_path"`

	DatabaseType     string `flag:"database_type"`
	DatabaseHostname string `flag:"database_hostname"`
	DatabaseName     string `flag:"database_name"`
	DatabaseUsername string `flag:"database_username"`
	DatabasePassword string `flag:"database_password"`

	DatabaseFile string `flag:"database_file"`

	EnableHTTP2 bool   `flag:"enable_http2" env:"SPEEDTEST_ENABLED_HTTP2"`
	EnableTLS   bool   `flag:"enable_tls"`
	TLSCertFile string `flag:"tls_cert_file"`
	TLSKeyFile  string `flag:"tls_key_file"`
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

const envPrefix = "SPEEDTEST_"

func Load() (*Config, error) {
	var k = koanf.New(".")

	configPathFlag := flag.String("c", "", "config file to be used, defaults to settings.toml in the same directory")

	// fill and map struct fields to flags
	if err := flagsfiller.New(
		flagsfiller.WithEnv(envPrefix),
		flagsfiller.NoSetFromEnv(),
	).Fill(flag.CommandLine, config); err != nil {
		slog.Error("filler flag from struct", slog.Any("error", err))
		return nil, err
	}
	flag.Parse()
	configPath := *configPathFlag
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

	if err := k.Load(env.Provider(envPrefix, ".", func(s string) string {
		ret := strings.TrimPrefix(s, envPrefix)
		ret = strings.ToLower(ret)
		return ret
	}), nil); err != nil {
		slog.Error("loading from env", slog.Any("error", err))
		return nil, err
	}

	if err := k.Load(providerWithValue(flag.CommandLine, ".", nil), nil); err != nil {
		slog.Error("loading from flag", slog.Any("error", err))
		return nil, err
	}
	if err := k.UnmarshalWithConf("", config, koanf.UnmarshalConf{Tag: "flag"}); err != nil {
		slog.Error("unmarshal to config", slog.Any("error", err))
		return nil, err
	}
	return config, nil
}

func LoadedConfig() *Config {
	return config
}
