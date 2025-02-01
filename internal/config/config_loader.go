package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config holds the application's configuration structure.
type Config struct {
	Kafka struct {
		BootstrapServers []string `yaml:"bootstrap_servers"`
		GroupID          string   `yaml:"group_id"`
		Topic            string   `yaml:"topic"`
		Security         struct {
			Protocol                        string `yaml:"protocol"`
			SASLMechanisms                  string `yaml:"sasl_mechanisms"`
			Username                        string `yaml:"username"`
			Password                        string `yaml:"password"`
			SSLCALocation                   string `yaml:"ssl_ca_location"`
			SSLKeyLocation                  string `yaml:"ssl_key_location"`
			SSLCertificateLocation          string `yaml:"ssl_certificate_location"`
			EndpointIdentificationAlgorithm string `yaml:"endpoint_identification_algorithm"`
		} `yaml:"security"`
		AutoOffsetReset string `yaml:"auto_offset_reset"`
		PollIntervalMs  int    `yaml:"poll_interval_ms"`
	} `yaml:"kafka"`

	Application struct {
		WebhookURL           string `yaml:"webhook_url"`
		TransferWebhookURL   string `yaml:"transfer_webhook_url"`
		WalletListURL        string `yaml:"wallet_list_url"`
		WalletUpdateInterval int    `yaml:"wallet_update_interval"`
		UIDCharset           string `yaml:"uid_charset"`
		SolToUsdAPIURL       string `yaml:"sol_to_usd_api_url"`
		SaveMatches          string `yaml:"save_matches"`
	} `yaml:"application"`

	Interpreter struct {
		BitqueryToken      string `yaml:"bitquery_token"`
		Python             string `yaml:"python"`
		SwapDetectorScript string `yaml:"swap_detector_script"`
	} `yaml:"interpreter"`

	Metrics struct {
		FlushIntervalSeconds int    `yaml:"flush_interval_seconds"`
		CloudEndpoint        string `yaml:"cloud_endpoint"`
		MaxMetricsCacheSize  int    `yaml:"max_metrics_cache_size"`
	} `yaml:"metrics"`
}

// LoadConfig reads and parses a YAML configuration file into a Config struct.
func LoadConfig(filepath string) (*Config, error) {
	// Read the config file
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the YAML data into the Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and return the configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateConfig ensures required fields in the configuration are populated.
func validateConfig(cfg *Config) error {
	// Validate Kafka configuration
	if len(cfg.Kafka.BootstrapServers) == 0 {
		return fmt.Errorf("kafka.bootstrap_servers are required")
	}
	if cfg.Kafka.GroupID == "" {
		return fmt.Errorf("kafka.group_id is required")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("kafka.topic is required")
	}
	if cfg.Kafka.Security.Protocol == "" {
		return fmt.Errorf("kafka.security.protocol is required")
	}
	if cfg.Kafka.Security.SASLMechanisms == "" {
		return fmt.Errorf("kafka.security.sasl_mechanisms is required")
	}
	if cfg.Kafka.Security.Username == "" {
		return fmt.Errorf("kafka.security.username is required")
	}
	if cfg.Kafka.Security.Password == "" {
		return fmt.Errorf("kafka.security.password is required")
	}
	if cfg.Kafka.Security.SSLCALocation == "" {
		return fmt.Errorf("kafka.security.ssl_ca_location is required")
	}
	if cfg.Kafka.Security.SSLKeyLocation == "" {
		return fmt.Errorf("kafka.security.ssl_key_location is required")
	}
	if cfg.Kafka.Security.SSLCertificateLocation == "" {
		return fmt.Errorf("kafka.security.ssl_certificate_location is required")
	}

	// Validate Application configuration
	if cfg.Application.WebhookURL == "" {
		return fmt.Errorf("application.webhook_url is required")
	}
	if cfg.Application.TransferWebhookURL == "" {
		return fmt.Errorf("application.transfer_webhook_url is required")
	}
	if cfg.Application.WalletListURL == "" {
		return fmt.Errorf("application.wallet_list_url is required")
	}
	if cfg.Application.WalletUpdateInterval <= 0 {
		return fmt.Errorf("application.wallet_update_interval must be greater than 0")
	}
	if cfg.Application.UIDCharset == "" {
		return fmt.Errorf("application.uid_charset is required")
	}
	if cfg.Application.SolToUsdAPIURL == "" {
		return fmt.Errorf("application.sol_to_usd_api_url is required")
	}
	if cfg.Application.SaveMatches == "" {
		return fmt.Errorf("application.save_matches is required")
	}

	// Validate Interpreter configuration
	if cfg.Interpreter.BitqueryToken == "" {
		return fmt.Errorf("interpreter.bitquery_token is required")
	}
	if cfg.Interpreter.Python == "" {
		return fmt.Errorf("interpreter.python is required")
	}
	if cfg.Interpreter.SwapDetectorScript == "" {
		return fmt.Errorf("interpreter.swap_detector_script is required")
	}

	// Validate Metrics configuration
	if cfg.Metrics.FlushIntervalSeconds <= 0 {
		return fmt.Errorf("metrics.flush_interval_seconds must be greater than 0")
	}
	if cfg.Metrics.CloudEndpoint == "" {
		return fmt.Errorf("metrics.cloud_endpoint is required")
	}
	if cfg.Metrics.MaxMetricsCacheSize <= 0 {
		return fmt.Errorf("metrics.max_metrics_cache_size must be greater than 0")
	}

	return nil
}
