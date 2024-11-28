// Copyright (c) 2024 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package cmd

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/retr0h/osapi/internal/config"
)

var (
	appConfig  config.Config
	appFs      = afero.NewOsFs()
	logger     = slog.New(slog.NewTextHandler(os.Stdout, nil))
	jsonOutput bool
)

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
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Enable JSON output")

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
		logFatal("failed to read config", err, "osapiFile", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		logFatal("failed to unmarshal config", err, "osapiFile", viper.ConfigFileUsed())
	}

	err := config.Validate(&appConfig)
	if err != nil {
		logFatal("validation failed", err, "osapiFile", viper.ConfigFileUsed())
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

	if jsonOutput {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
}
