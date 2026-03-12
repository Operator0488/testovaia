package config

import "os"

// InitOptions holds parameters for Init.
type InitOptions struct {
	ConfigPath string
	FileName   string
}

// InitOption is a functional option that takes current options and returns updated ones.
type InitOption func(InitOptions) InitOptions

// WithConfigPath sets the directory where the config file is searched.
func WithConfigPath(path string) InitOption {
	return func(o InitOptions) InitOptions {
		o.ConfigPath = path
		return o
	}
}

// WithFileName sets the config file name (without extension, e.g. "env" or "envs.v2").
func WithFileName(name string) InitOption {
	return func(o InitOptions) InitOptions {
		o.FileName = name
		return o
	}
}

func applyInitOptions(opts ...InitOption) InitOptions {
	fileName, _ := os.LookupEnv("CONFIG_NAME")
	if len(fileName) == 0 {
		fileName = defaultFileName
	}
	o := InitOptions{
		ConfigPath: "./bootstrap",
		FileName:   fileName,
	}
	for _, opt := range opts {
		if opt != nil {
			o = opt(o)
		}
	}
	return o
}
