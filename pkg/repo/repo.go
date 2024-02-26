package repo

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
)

type Repo struct {
	RepoRoot string
	Config   *Config
}

func LoadRepoRootFromEnv(repoRoot string) (string, error) {
	if repoRoot != "" {
		return repoRoot, nil
	}
	repoRoot = os.Getenv(RootPathEnvVar)
	var err error
	if len(repoRoot) == 0 {
		repoRoot, err = homedir.Expand(DefaultPathRoot)
	}
	return repoRoot, err
}

func Load(repoRoot string) (*Repo, error) {
	repoRoot, err := LoadRepoRootFromEnv(repoRoot)
	if err != nil {
		return nil, err
	}
	cfg, err := LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	repo := &Repo{
		RepoRoot: repoRoot,
		Config:   cfg,
	}

	return repo, nil
}

func CheckWritable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := filepath.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		_ = fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesn't exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

func writeConfigWithEnv(cfgPath string, config any) error {
	if err := writeConfig(cfgPath, config); err != nil {
		return err
	}
	// write back environment variables first
	// TODO: wait viper support read from environment variables
	if err := readConfigFromFile(cfgPath, config); err != nil {
		return errors.Wrapf(err, "failed to read cfg from environment")
	}
	if err := writeConfig(cfgPath, config); err != nil {
		return err
	}
	return nil
}

func writeConfig(cfgPath string, config any) error {
	raw, err := MarshalConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, []byte(raw), 0755); err != nil {
		return err
	}

	return nil
}

func MarshalConfig(config any) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	e := toml.NewEncoder(buf)
	e.SetIndentTables(true)
	e.SetArraysMultiline(true)
	err := e.Encode(config)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r *Repo) Flush() error {
	if err := writeConfigWithEnv(path.Join(r.RepoRoot, CfgFileName), r.Config); err != nil {
		return errors.Wrap(err, "failed to write config")
	}
	return nil
}
