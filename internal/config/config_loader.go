package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

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
		WebhookURL                  string `yaml:"webhook_url"`
		WalletListURL               string `yaml:"wallet_list_url"`
		WalletUpdateIntervalSeconds int    `yaml:"wallet_update_interval"`
		UIDCharset                  string `yaml:"uid_charset"`
		SolToUsdAPIURL              string `yaml:"sol_to_usd_api_url"`
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

func LoadConfig(filepath string) (*Config, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
		return nil, err
	}

	return &cfg, nil
}
