package repo

const (
	// DefaultPathName is the default config dir name
	DefaultPathName = ".faucet"

	// DefaultPathRoot is the path to the default config dir location.
	DefaultPathRoot = "~/" + DefaultPathName

	// EnvDir is the environment variable used to change the path root.
	EnvDir = "FAUCET_PATH"

	LogsDirName = "logs"

	// CfgFileName is config name
	CfgFileName = "faucet.toml"

	// API name
	APIName = "api"

	RootPathEnvVar = "FAUCET_ROOT_PATH"
)
