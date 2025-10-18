package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string        `yaml:"env" envDefault:"development"`
	Secret      string        `yaml:"secret" envDefault:"secret"`
	StoragePath string        `yaml:"storage_path"`
	GRPC        GRPCConfig    `yaml:"grpc"`
	Kafka       KafkaConfig   `yaml:"kafka"`
	Metrics     MetricsConfig `yaml:"metrics"`
}

type GRPCConfig struct {
	Port    int           `env:"port" envDefault:"50051"`
	Timeout time.Duration `env:"timeout" envDefault:"5s"`
}

type KafkaConfig struct {
	Brokers  []string `yaml:"brokers"`
	Topic    string   `yaml:"topic"`
	GroupID  string   `yaml:"group_id" envDefault:"user-service-group"`
	DialAddr string   `yaml:"dial_addr" envDefault:"kafka:9092"`
}

type MetricsConfig struct {
	Port int    `yaml:"port" envDefault:"8082"`
	Host string `yaml:"host" envDefault:"localhost"`
}

func MustLoad() *Config {
	path := fetchConfigPath()

	if path == "" {
		panic("config path is not provided")
	}

	return MustLoadByPath(path)
}

func MustLoadByPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("Failed to read config: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
