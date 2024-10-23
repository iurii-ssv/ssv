package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/cobra"
)

// Args expose available global args for cli command
type Args struct {
	ConfigPath      string
	ShareConfigPath string
}

// Global expose available global config for cli command
type Global struct {
	LogLevel       string `yaml:"LogLevel" env:"LOG_LEVEL" env-default:"info" env-description:"Defines logger's log level"`
	LogFormat      string `yaml:"LogFormat" env:"LOG_FORMAT" env-default:"console" env-description:"Defines logger's encoding, valid values are 'json' and 'console'(default)"`
	LogLevelFormat string `yaml:"LogLevelFormat" env:"LOG_LEVEL_FORMAT" env-default:"capitalColor" env-description:"Defines logger's level format, valid values are 'capitalColor' (default), 'capital' or 'lowercase'"`
	LogFilePath    string `yaml:"LogFilePath" env:"LOG_FILE_PATH" env-default:"./data/debug.log" env-description:"Defines a file path to write logs into"`
	LogFileSize    int    `yaml:"LogFileSize" env:"LOG_FILE_SIZE" env-default:"500" env-description:"Defines a file size in megabytes to rotate logs"`
	LogFileBackups int    `yaml:"LogFileBackups" env:"LOG_FILE_BACKUPS" env-default:"3" env-description:"Defines a number of backups to keep when rotating logs"`
}

// ProcessArgs processes and handles CLI arguments
func ProcessArgs(cfg interface{}, a *Args, cmd *cobra.Command) {
	configFlag := "config"
	cmd.PersistentFlags().StringVarP(&a.ConfigPath, configFlag, "c", "./config/config.yaml", "Path to configuration file")
	_ = cmd.MarkFlagRequired(configFlag)

	shareConfigFlag := "share-config"
	cmd.PersistentFlags().StringVarP(&a.ShareConfigPath, shareConfigFlag, "s", "", "Path to local share configuration file")
	_ = cmd.MarkFlagRequired(shareConfigFlag)

	envHelp, _ := cleanenv.GetDescription(cfg, nil)
	cmd.SetUsageTemplate(envHelp + "\n" + cmd.UsageTemplate())
}

func Prepare(cfg interface{}, a *Args) error {
	if a.ConfigPath != "" {
		err := cleanenv.ReadConfig(a.ConfigPath, cfg)
		if err != nil {
			return fmt.Errorf("could not read config: %w", err)
		}
		return nil
	}
	if a.ShareConfigPath != "" {
		err := cleanenv.ReadConfig(a.ShareConfigPath, cfg)
		if err != nil {
			return fmt.Errorf("could not read share config: %w", err)
		}
		return nil
	}

	// When no config file is provided fall back to env vars + default annotations defined on cfg struct.
	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		return fmt.Errorf("could not set up config based on environment variables and struct defaults: %w", err)
	}
	return nil
}
