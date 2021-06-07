package repo

import (
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
)

const (
	configName = "faucet.toml"
)

type Config struct {
	RepoRoot string
	Ether    Ether   `toml:"ether" json:"ether"`
	Bxh      Bxh     `toml:"bxh" json:"bxh"`
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

type Ether struct {
	Addr            string `toml:"addr" json:"addr"`
	Name            string `toml:"name" json:"name"`
	ContractAddress string `mapstructure:"contract_address" json:"contract_address"`
	KeyPath         string `mapstructure:"key_path" json:"key_path"`
	Password        string `toml:"password" json:"password"`
	MinConfirm      uint64 `mapstructure:"min_confirm" json:"min_confirm"`
}
type Bxh struct {
	BxhAddr     string `mapstructure:"bxh_addr" json:"bxh_addr"`
	BxhKeyPath  string `mapstructure:"bxh_key_path" json:"bxh_key_path"`
	BxhPassword string `mapstructure:"bxh_password" json:"bxh_password"`
	MinConfirm  uint64 `mapstructure:"min_confirm" json:"min_confirm"`
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
