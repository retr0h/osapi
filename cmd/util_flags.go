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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// registerDatabaseFlags defines and binds the flags related to database configuration.
func registerDatabaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().
		StringP("driver-name", "t", "sqlite", "Name of the database driver to use")
	cmd.PersistentFlags().
		StringP("dsn", "s", ":memory:?_journal=WAL&_timeout=5000&_fk=true", "The data source name (DSN) for the database connection")
	cmd.PersistentFlags().
		IntP("max-open-conns", "o", 1, "The maximum number of open connections to the database")
	cmd.PersistentFlags().
		IntP("max-idle-conns", "i", 1, "The maximum number of idle connections in the pool")

	_ = viper.BindPFlag("database.driver_name", cmd.PersistentFlags().Lookup("driver-name"))
	_ = viper.BindPFlag("database.data_source_name", cmd.PersistentFlags().Lookup("dsn"))
	_ = viper.BindPFlag(
		"database.max_open_conns",
		cmd.PersistentFlags().Lookup("max-open-conns"),
	)
	_ = viper.BindPFlag(
		"database.max_idle_conns",
		cmd.PersistentFlags().Lookup("max-idle-conns"),
	)
}
