package repo

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	configName = "faucet.toml"
)

type Config struct {
	RepoRoot string
	Axiom    AXIOM   `toml:"axiom" json:"axiom"`
	Network  Network `toml:"network" json:"network"`
	Log      Log     `toml:"log" json:"log"`
}

// Log are config about log
type Log struct {
	Dir          string    `toml:"dir" json:"dir"`
	Filename     string    `toml:"filename" json:"filename"`
	ReportCaller bool      `mapstructure:"report_caller"`
	Level        string    `toml:"level" json:"level"`
	Module       LogModule `toml:"module" json:"module"`
}
type LogModule struct {
	ApiServer string `mapstructure:"api_server" toml:"api_server" json:"api_server"`
}

type AXIOM struct {
	AxiomAddr    string `mapstructure:"axiom_addr" json:"axiom_addr"`
	AxiomKeyPath string `mapstructure:"axiom_key_path" json:"axiom_key_path"`
	MinConfirm   uint64 `mapstructure:"min_confirm" json:"min_confirm"`
}

type Network struct {
	Port string `mapstructure:"port" json:"port"`
}

func defaultConfig() *Config {
	return &Config{}
}

func UnmarshalConfig(configRoot string) (*Config, error) {
	viper.SetConfigFile(filepath.Join(configRoot, configName))
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ETHER")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := defaultConfig()

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	config.RepoRoot = configRoot

	return config, nil
}
