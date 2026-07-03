package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host              string `yaml:"host"`
	Port              int    `yaml:"port"`
	ReadTimeoutSec    int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSec   int    `yaml:"write_timeout_seconds"`
}

type QueueConfig struct {
	Workers        int `yaml:"workers"`
	MaxQueueSize   int `yaml:"max_queue_size"`
	JobTimeoutSec  int `yaml:"job_timeout_seconds"`
	MaxRetries     int `yaml:"max_retries"`
}

type TorConfig struct {
	Enabled   bool   `yaml:"enabled"`
	SocksHost string `yaml:"socks_host"`
	SocksPort int    `yaml:"socks_port"`
}

type AIConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Key      string `yaml:"key"`
}

type RateLimitConfig struct {
	RequestsPerMin int `yaml:"requests_per_minute"`
	Burst          int `yaml:"burst"`
}

type CollectorItem struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key,omitempty"`
	ID      string `yaml:"id,omitempty"`
	Secret  string `yaml:"secret,omitempty"`
}

type GeoIPConfig struct {
	CityDBPath string `yaml:"city_db_path"`
	ASNDBPath  string `yaml:"asn_db_path"`
	AccountID  int    `yaml:"account_id"`
	LicenseKey string `yaml:"license_key"`
}

type WebSearchConfig struct {
	Enabled        bool   `yaml:"enabled"`
	GoogleAPIKey   string `yaml:"google_api_key"`
	GoogleEngineID string `yaml:"google_engine_id"`
	BraveAPIKey    string `yaml:"brave_api_key"`
}

type CollectorsConfig struct {
	GeoIP       CollectorItem `yaml:"geoip"`
	DNS          CollectorItem `yaml:"dns"`
	Whois        CollectorItem `yaml:"whois"`
	Crtsh        CollectorItem `yaml:"crtsh"`
	Subdomains   CollectorItem `yaml:"subdomains"`
	DNSDumpster  CollectorItem `yaml:"dnsdumpster"`
	Shodan       CollectorItem `yaml:"shodan"`
	Censys       CollectorItem `yaml:"censys"`
	VirusTotal   CollectorItem `yaml:"virustotal"`
	GreyNoise    CollectorItem `yaml:"greynoise"`
	Github       CollectorItem `yaml:"github"`
	EmailRep     CollectorItem `yaml:"emailrep"`
	URLScan      CollectorItem `yaml:"urlscan"`
	Ethereum     CollectorItem `yaml:"ethereum"`
	Tron         CollectorItem `yaml:"tron"`
	Bitcoin      CollectorItem `yaml:"bitcoin"`
	SearchEngine CollectorItem `yaml:"searchengine"`
	Wayback      CollectorItem `yaml:"wayback"`
	Pastebin     CollectorItem `yaml:"pastebin"`
	Onion        CollectorItem `yaml:"onion"`
	Social       CollectorItem `yaml:"social"`
	ThreatIntel  CollectorItem `yaml:"threatintel"`
	AISummary    CollectorItem `yaml:"ai_summary"`
}

type APIAuthConfig struct {
	RequireKey bool     `yaml:"require_key"`
	Keys       []string `yaml:"keys"`
}

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Queue      QueueConfig      `yaml:"queue"`
	Tor        TorConfig        `yaml:"tor"`
	AI         AIConfig         `yaml:"ai"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	GeoIP      GeoIPConfig      `yaml:"geoip"`
	WebSearch  WebSearchConfig  `yaml:"websearch"`
	Collectors CollectorsConfig `yaml:"collectors"`
	APIAuth    APIAuthConfig    `yaml:"api_keys"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeoutSec:  30,
			WriteTimeoutSec: 120,
		},
		Queue: QueueConfig{
			Workers:       500,
			MaxQueueSize:   10000,
			JobTimeoutSec:  30,
			MaxRetries:     3,
		},
		Tor: TorConfig{
			Enabled:   false,
			SocksHost: "127.0.0.1",
			SocksPort: 9050,
		},
		AI: AIConfig{
			Provider: "groq",
			Model:    "llama-3.3-70b-versatile",
		},
		WebSearch: WebSearchConfig{
			Enabled: true,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMin: 10,
			Burst:          3,
		},
		GeoIP: GeoIPConfig{
			CityDBPath: "data/GeoLite2-City.mmdb",
			ASNDBPath:  "data/GeoLite2-ASN.mmdb",
		},
		APIAuth: APIAuthConfig{
			RequireKey: false,
		},
	}
}

func (c *Config) ResolveEnv() {
	resolve := func(v string) string {
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}")
			if e := os.Getenv(key); e != "" {
				return e
			}
		}
		return v
	}
	c.AI.Key = resolve(c.AI.Key)
	c.Collectors.Shodan.Key = resolve(c.Collectors.Shodan.Key)
	c.Collectors.Censys.Key = resolve(c.Collectors.Censys.Key)
	c.Collectors.VirusTotal.Key = resolve(c.Collectors.VirusTotal.Key)
	c.Collectors.GreyNoise.Key = resolve(c.Collectors.GreyNoise.Key)
	c.Collectors.Github.Key = resolve(c.Collectors.Github.Key)
	c.Collectors.EmailRep.Key = resolve(c.Collectors.EmailRep.Key)
	c.Collectors.URLScan.Key = resolve(c.Collectors.URLScan.Key)
	c.Collectors.Ethereum.Key = resolve(c.Collectors.Ethereum.Key)
	c.Collectors.Tron.Key = resolve(c.Collectors.Tron.Key)
	c.GeoIP.LicenseKey = resolve(c.GeoIP.LicenseKey)
	c.WebSearch.GoogleAPIKey = resolve(c.WebSearch.GoogleAPIKey)
	c.WebSearch.GoogleEngineID = resolve(c.WebSearch.GoogleEngineID)
	c.WebSearch.BraveAPIKey = resolve(c.WebSearch.BraveAPIKey)
}

var apiKeyMapping = map[string]string{
	"groq":         "GROQ_API_KEY",
	"github":       "GITHUB_TOKEN",
	"shodan":       "SHODAN_KEY",
	"Censysapi":    "CENSYS_KEY",
	"VT_key":       "VT_KEY",
	"urlscan":      "URLSCAN_KEY",
	"eth-api-key":  "ETHERSCAN_KEY",
	"tron-api-key": "TRON_KEY",
}

func LoadEnvFrom(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open apifile: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "'\"")
		val = strings.TrimSpace(val)
		if env, ok := apiKeyMapping[key]; ok {
			os.Setenv(env, val)
			continue
		}
		baseKey := strings.TrimRight(key, "0123456789")
		if baseKey != key {
			if env, ok := apiKeyMapping[baseKey]; ok {
				os.Setenv(env, val)
			}
		}
	}
	return scanner.Err()
}

func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open .env: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "'\"")
		val = strings.TrimSpace(val)
		if val != "" {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func Load(path string) (*Config, error) {
	cfg := Default()

	apiTxtPath := "api.txt"
	if env := os.Getenv("NEBULA_API_TXT"); env != "" {
		apiTxtPath = env
	}
	if err := LoadEnvFrom(apiTxtPath); err != nil {
		return nil, err
	}
	if err := LoadDotEnv(".env"); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.ResolveEnv()
	return cfg, nil
}
