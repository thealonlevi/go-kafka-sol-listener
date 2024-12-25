package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration structure for the application.
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
	} `yaml:"kafka"`
}

// LoadConfig loads the configuration from the specified YAML file.
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
