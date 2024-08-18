/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/retr0h/osapi/internal/config"
)

var (
	appConfig config.Config
	logger    *slog.Logger
)

func logFatal(message string, logGroup any) {
	initLogger()
	logger.Error(
		message,
		logGroup,
	)

	os.Exit(1)
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "osapi",
	Short: "A CRUD API for managing Linux systems.",
	Long: `A CRUD API for managing Linux systems, responsible for ensuring that
the system's configuration matches the desired state.

┌─┐┌─┐┌─┐┌─┐┬
│ │└─┐├─┤├─┘│
└─┘└─┘┴ ┴┴  ┴

https://github.com/retr0h/osapi
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable or disable debug mode")
	rootCmd.PersistentFlags().
		StringP("osapi-file", "f", "osapi.yaml", "Path to config file")

	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("osapiFile", rootCmd.PersistentFlags().Lookup("osapi-file"))
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("osapi")
	viper.SetConfigFile(viper.GetString("osapiFile"))

	if err := viper.ReadInConfig(); err != nil {
		logFatal(
			"failed to read config",
			slog.Group("",
				slog.String("osapiFile", viper.ConfigFileUsed()),
				slog.String("err", err.Error()),
			),
		)
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		logFatal(
			"failed to unmarshal config",
			slog.Group("",
				slog.String("osapiFile", viper.ConfigFileUsed()),
				slog.String("err", err.Error()),
			),
		)
	}
}

func initLogger() {
	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}

	logger = slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      logLevel,
			TimeFormat: time.Kitchen,
			NoColor:    !term.IsTerminal(int(os.Stdout.Fd())),
		}),
	)
}
