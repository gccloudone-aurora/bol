package util

import (
	"fmt"

	validator "github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type TLSClientConfig struct {
	Insecure bool   `mapstructure:"insecure" validate:"required"`
	CaData   string `mapstructure:"caData" validate:"required"`
}

type ManualK8sAuth struct {
	Host            string          `mapstructure:"host" validate:"required"`
	BearerToken     string          `mapstructure:"bearerToken" validate:"required"`
	TLSClientConfig TLSClientConfig `mapstructure:"tlsClientConfig" validate:"required"`
}

type KubernetesAuth struct {
	Method         string        `mapstructure:"method" validate:"required,oneof=incluster kubeconfigPath manual"`
	KubeconfigPath string        `mapstructure:"kubeconfigPath" validate:"required_if=Method kubeconfigPath"`
	Manual         ManualK8sAuth `mapstructure:"manual" validate:"required_if=Method manual"`
}

type Cluster struct {
	Name                              string         `mapstructure:"name" validate:"required"`
	Subscription                      string         `mapstructure:"subscription" validate:"required"`
	KubecostURL                       string         `mapstructure:"kubecostURL" validate:"required"`
	KubecostURlAttachAzurebearerToken bool           `mapstructure:"kubecostURlAttachAzurebearerToken" validate:"required"`
	KubernetesAuth                    KubernetesAuth `mapstructure:"kubernetesAuth" validate:"required"`
}

type AzureConfig struct {
	StorageAccountName          string `mapstructure:"storageAccountName" validate:"required"`
	StorageAccountContainerName string `mapstructure:"storageAccountContainerName" validate:"required"`
	UseAzureCliCredentials      bool   `mapstructure:"useAzureCliCredentials"`
}

type ArtifactRepository struct {
	Provider string      `mapstructure:"provider" validate:"required,oneof=azure"`
	Azure    AzureConfig `mapstructure:"azure" validate:"required_if=Provider azure"`
}

type Config struct {
	MaximumReportingWindowInDays int                `mapstructure:"maximumReportingWindowInDays" validate:"required,gte=1,lte=30"`
	FileNameSuffix               string             `mapstructure:"fileNameSuffix"`
	ArtifactRepository           ArtifactRepository `mapstructure:"artifactRepository" validate:"required"`
	Clusters                     []Cluster          `mapstructure:"clusters" validate:"required"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (*Config, error) {
	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/bol/") // path to look for the config file in
		viper.AddConfigPath(".")         // optionally look for config in the working directory
	}

	viper.SetDefault("maximumReportingWindowInDays", 30)
	viper.SetDefault("fileNameSuffix", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("Config file not found: %v", err)
		} else {
			return nil, fmt.Errorf("Config file was found but another error was produced: %v", err)
		}
	}

	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode into struct, %v", err)
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, err
	}

	return &config, err
}
