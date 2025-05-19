package config

import (
	"fmt"
	"os"

	"github.com/shoksin/financesBot/internal/logger"
	"gopkg.in/yaml.v3"
)

const configFile = "data/config.yaml"

type Config struct {
	Token                       string   `yaml:"token"`
	MainCurrency                string   `yaml:"main_currency"`
	CurrenciesName              []string `yaml:"currencies_name"`
	CurrenciesUpdatePeriod      int64    `yaml:"currencies_update_period"`       // Периодичность обновления курсов валют (в минутах).
	CurrenciesUpdateCachePeriod int64    `yaml:"currencies_update_cache_period"` // Периодичность кэширования курсов валют из базы данных (в минутах).
	ConnectionStringDB          string   `yaml:"connection_string_db"`
	KafkaTopic                  string   `yaml:"kafka_topic"`
	BrokersList                 []string `yaml:"brokers_list"` // Список адресов брокеров сообщений (адрес Kafka).
}

type Service struct {
	config Config
}

func New() (*Service, error) {
	s := &Service{}

	rawYAML, err := os.ReadFile(configFile)
	if err != nil {
		logger.Error("Error read config file", "err", err)
		return nil, fmt.Errorf("reading config error: %v", err)
	}

	err = yaml.Unmarshal(rawYAML, &s.config)
	if err != nil {
		logger.Error("Error to unmarshal config data", "err", err)
		return nil, fmt.Errorf("unmarhaling config error: %v", err)
	}

	return s, nil
}

func (s *Service) Token() string {
	return s.config.Token
}

func (s *Service) GetConfig() Config {
	return s.config
}
