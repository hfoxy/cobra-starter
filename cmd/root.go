package cmd

// most of this code is from: https://github.com/carolynvs/stingoftheviper/blob/main/main.go

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/hfoxy/cobra-starter/flags"
	"github.com/hfoxy/cobra-starter/logging"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configBaseRootCmd = &cobra.Command{
		Use:   "cobra-starter",
		Short: "starter for cobra commands",
	}
	configDefaultConfigFilename      = "cobra-starter"
	configEnvPrefix                  = "COBRA_STARTER"
	configReplaceHyphenWithCamelCase = false
)

var configParams []string

type CommandConfig struct {
	BaseCommand                *cobra.Command
	ConfigFileName             string
	EnvPrefix                  string
	ReplaceHyphenWithCamelCase bool
	DisableDefaultFlags        bool
	DisableConfigFlags         bool
	RootFlags                  RootFlags
	Commands                   []CommandAdder
	AdditionalConfigs          []string
}

type RootFlags func(rootCmd *cobra.Command) error
type CommandAdder func(rootCmd *cobra.Command) (*cobra.Command, error)

func NewRootCommand(config CommandConfig) (*cobra.Command, error) {
	if config.ConfigFileName == "" {
		return nil, fmt.Errorf("config file name must be specified")
	}

	configDefaultConfigFilename = config.ConfigFileName
	configEnvPrefix = config.EnvPrefix
	configReplaceHyphenWithCamelCase = config.ReplaceHyphenWithCamelCase

	rootCmd := configBaseRootCmd
	if config.BaseCommand != nil {
		rootCmd = config.BaseCommand
	}

	configs := config.AdditionalConfigs
	if configs == nil {
		configs = []string{}
	}

	existingPPRE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := initializeConfig(cmd, configs...)
		if err != nil {
			return fmt.Errorf("unable to initialize config: %v", err)
		}

		if existingPPRE != nil {
			return existingPPRE(cmd, args)
		} else {
			return nil
		}
	}

	var err error
	if !config.DisableDefaultFlags {
		err = addRootFlags(rootCmd)
		if err != nil {
			return nil, err
		}
	}

	if !config.DisableConfigFlags {
		err = addConfigFlags(rootCmd)
		if err != nil {
			return nil, err
		}
	}

	if config.RootFlags != nil {
		if err = config.RootFlags(rootCmd); err != nil {
			return nil, err
		}
	}

	if config.Commands != nil {
		for _, command := range config.Commands {
			var cmd *cobra.Command
			if cmd, err = command(rootCmd); err != nil {
				return nil, err
			}

			cmdPPRE := cmd.PersistentPreRunE
			cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				err = initializeConfig(cmd, configs...)
				if err != nil {
					return fmt.Errorf("unable to initialize config: %v", err)
				}

				if cmdPPRE != nil {
					return cmdPPRE(cmd, args)
				} else {
					return nil
				}
			}

			rootCmd.AddCommand(cmd)
		}
	}

	return rootCmd, nil
}

func addRootFlags(cmd *cobra.Command) error {
	// Define cobra flags, the default value has the lowest (least significant) precedence

	cmd.PersistentFlags().BoolVarP(&flags.DebugEnabled, "debug", "d", flags.DebugEnabled, "Enable debug logging")
	cmd.PersistentFlags().StringVarP(&logging.LogFormat, "log-format", "f", logging.LogFormat, "Default log format (text, json)")
	cmd.PersistentFlags().StringVarP(&logging.LogOutputs, "log-outputs", "o", logging.LogOutputs, "Comma separated list of log outputs (stdout,file). Can specify format for each output (stdout:text,file:json) (stdout, file)")
	return nil
}

func addConfigFlags(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVar(&configParams, "configs", []string{}, "Additional config files to load")
	return nil
}

func initializeConfig(cmd *cobra.Command, configs ...string) error {
	if configParams != nil && len(configParams) > 0 {
		configs = append(configs, configParams...)
	}

	notifyFunc := func(v fsnotify.Event) {}
	return initializeWatchConfig(cmd, notifyFunc, configs...)
}

func initializeWatchConfig(cmd *cobra.Command, onChange func(event fsnotify.Event), configs ...string) error {
	if err := initSpecificConfig(cmd, onChange, configDefaultConfigFilename); err != nil {
		return fmt.Errorf("unable to load config '%s': %v", configDefaultConfigFilename, err)
	}

	for _, config := range configs {
		if err := initSpecificConfig(cmd, onChange, config); err != nil {
			return fmt.Errorf("unable to load config '%s': %v", config, err)
		}
	}

	logging.Init()
	return nil
}

func initSpecificConfig(cmd *cobra.Command, onChange func(event fsnotify.Event), filename string) error {
	v := viper.New()
	v.SetConfigName(filename)

	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	if filename == configDefaultConfigFilename {
		if configEnvPrefix != "" {
			v.SetEnvPrefix(configEnvPrefix)
		}

		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
	}

	v.OnConfigChange(onChange)

	// Bind the current command's flags to viper
	bindFlags(cmd, v)
	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name
		// If using camelCase in the config file, replace hyphens with a camelCased string.
		// Since viper does case-insensitive comparisons, we don't need to bother fixing the case, and only need to remove the hyphens.
		if configReplaceHyphenWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}

		// ensure keys like postgres.host are converted to postgres-host
		for {
			// Apply the viper config value to the flag when the flag is not set and viper has a value
			if !f.Changed && v.IsSet(configName) {
				val := v.Get(configName)
				cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}

			if strings.Contains(configName, "-") {
				configName = strings.Replace(configName, "-", ".", 1)
			} else {
				break
			}
		}
	})
}
