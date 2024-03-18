package repo

import (
	"bytes"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Duration time.Duration

func (d *Duration) ToDuration() time.Duration {
	return time.Duration(*d)
}

func (d *Duration) String() string {
	return time.Duration(*d).String()
}

type Config struct {
	Axiom    AXIOM    `toml:"axiom" json:"axiom"`
	Network  Network  `toml:"network" json:"network"`
	Log      Log      `toml:"log" json:"log"`
	Scrapper Scrapper `toml:"scrapper" json:"scrapper"`
}

type Scrapper struct {
	ScrapperAddr string `mapstructure:"scrapper_addr" json:"scrapper_addr"`
}

// Log are config about log
type Log struct {
	Filename     string `toml:"filename" json:"filename"`
	ReportCaller bool   `mapstructure:"report_caller"`
	Level        string `toml:"level" json:"level"`

	EnableCompress   bool     `mapstructure:"enable_compress" toml:"enable_compress"`
	EnableColor      bool     `mapstructure:"enable_color" toml:"enable_color"`
	DisableTimestamp bool     `mapstructure:"disable_timestamp" toml:"disable_timestamp"`
	MaxAge           uint     `mapstructure:"max_age" toml:"max_age"`
	MaxSize          uint     `mapstructure:"max_size" toml:"max_size"`
	RotationTime     Duration `mapstructure:"rotation_time" toml:"rotation_time"`

	Module LogModule `toml:"module" json:"module"`
}
type LogModule struct {
	ApiServer string `mapstructure:"api_server" toml:"api_server" json:"api_server"`
	Global    string `mapstructure:"global" toml:"global" json:"global"`
}

type AXIOM struct {
	TestNetName  string  `mapstructure:"test_net_name" json:"test_net_name"`
	AxiomAddr    string  `mapstructure:"axiom_addr" json:"axiom_addr"`
	AxiomKeyPath string  `mapstructure:"axiom_key_path" json:"axiom_key_path"`
	Amount       float64 `mapstructure:"amount" json:"amount"`
	TweetAmount  float64 `mapstructure:"tweet_amount" json:"tweet_amount"`
	ClaimLimit   float64 `mapstructure:"claim_limit" json:"claim_limit"`
	GasLimit     uint64  `mapstructure:"gas_limit" json:"gas_limit"`
}

type Network struct {
	Port string `mapstructure:"port" json:"port"`
}

func DefaultConfig() *Config {
	return &Config{
		Axiom: AXIOM{
			TestNetName:  "Taurus",
			AxiomAddr:    "http://127.0.0.1:8881",
			AxiomKeyPath: "axiom.account.key",
			Amount:       100,
			TweetAmount:  200,
			ClaimLimit:   600,
			GasLimit:     21000,
		},
		Network: Network{
			Port: "8080",
		},
		Log: Log{
			Filename:         "faucet",
			ReportCaller:     false,
			Level:            "info",
			EnableCompress:   false,
			EnableColor:      true,
			DisableTimestamp: false,
			MaxAge:           30,
			MaxSize:          128,
			RotationTime:     Duration(24 * time.Hour),
			Module: LogModule{
				ApiServer: "info",
				Global:    "info",
			},
		},
		Scrapper: Scrapper{
			ScrapperAddr: "http://127.0.0.1:5000/tweetCheck",
		},
	}

}

func LoadConfig(repoRoot string) (*Config, error) {
	cfg, err := func() (*Config, error) {
		cfg := DefaultConfig()
		cfgPath := path.Join(repoRoot, CfgFileName)
		existConfig := fileutil.Exist(cfgPath)
		if !existConfig {
			err := os.MkdirAll(repoRoot, 0755)
			if err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}

			if err := writeConfigWithEnv(cfgPath, cfg); err != nil {
				return nil, errors.Wrap(err, "failed to build default config")
			}
		} else {
			if err := CheckWritable(repoRoot); err != nil {
				return nil, err
			}
			if err := readConfigFromFile(cfgPath, cfg); err != nil {
				return nil, err
			}
		}

		return cfg, nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}
	return cfg, nil
}

func readConfigFromFile(cfgFilePath string, config any) error {
	vp := viper.New()
	vp.SetConfigFile(cfgFilePath)
	vp.SetConfigType("toml")

	// only check types, viper does not have a strong type checking
	raw, err := os.ReadFile(cfgFilePath)
	if err != nil {
		return err
	}
	decoder := toml.NewDecoder(bytes.NewBuffer(raw))
	checker := reflect.New(reflect.TypeOf(config).Elem())
	if err := decoder.Decode(checker.Interface()); err != nil {
		var decodeError *toml.DecodeError
		if errors.As(err, &decodeError) {
			return errors.Errorf("check config formater failed from %s:\n%s", cfgFilePath, decodeError.String())
		}

		return errors.Wrapf(err, "check config formater failed from %s", cfgFilePath)
	}

	return readConfig(vp, config)
}

func readConfig(vp *viper.Viper, config any) error {
	vp.AutomaticEnv()
	vp.SetEnvPrefix("FAUCET")
	replacer := strings.NewReplacer(".", "_")
	vp.SetEnvKeyReplacer(replacer)

	err := vp.ReadInConfig()
	if err != nil {
		return err
	}

	if err := vp.Unmarshal(config, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		StringToTimeDurationHookFunc(),
		func(
			f reflect.Kind,
			t reflect.Kind,
			data any) (any, error) {
			if f != reflect.String || t != reflect.Slice {
				return data, nil
			}

			raw := data.(string)
			if raw == "" {
				return []string{}, nil
			}
			raw = strings.TrimPrefix(raw, ";")
			raw = strings.TrimSuffix(raw, ";")

			return strings.Split(raw, ";"), nil
		},
	))); err != nil {
		return err
	}

	return nil
}

func StringToTimeDurationHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(Duration(5)) {
			return data, nil
		}

		d, err := time.ParseDuration(data.(string))
		if err != nil {
			return nil, err
		}
		return Duration(d), nil
	}
}
