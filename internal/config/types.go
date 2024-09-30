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

package config

// Config represents the root structure of the YAML configuration file.
// This struct is used to unmarshal configuration data from Viper.
type Config struct {
	Client
	Server
	Queue
	Database
	// Debug enable or disable debug option set from CLI.
	Debug bool `mapstruture:"debug"`
}

// Client configuration settings.
type Client struct {
	// URL the client will connect to
	URL string `mapstructure:"url"`
}

// Server configuration settings.
type Server struct {
	// Port the server will bind to.
	Port int `mapstructure:"port"`
	// Security-related configuration, such as CORS settings.
	Security Security `mapstructure:"security"`
}

// Security represents the "security" configuration under the "server" section.
type Security struct {
	// CORS Cross-Origin Resource Sharing (CORS) settings for the server.
	CORS CORS `mapstructure:"cors"`
}

// CORS represents the CORS (Cross-Origin Resource Sharing) settings.
type CORS struct {
	// List of origins allowed to access the server (e.g., "foo").
	AllowOrigins []string `mapstructure:"allow_origins,omitempty"`
}

// Queue configuration settings.
type Queue struct {
	PollInterval `mapstructure:"poll_interval,omitempty"`
}

// PollInterval interval settings.
type PollInterval struct {
	// The interval at which the worker will poll for new tasks in the queue (in seconds).
	Seconds int `mapstructure:"seconds"`
}

// Database configuration settings.
type Database struct {
	// DriverName specifies the name of the database driver to use, such as "sqlite".
	DriverName string `mapstructure:"driver_name"`
	// DataSourceName defines the data source name (DSN) for the database connection.
	// The DSN format depends on the database driver used. For SQLite, this often
	// includes the file path and query parameters like journal mode and timeout settings.
	DataSourceName string `mapstructure:"data_source_name"`
	// Maximum number of open connections to the database
	MaxOpenConns int `mapstructure:"max_open_conns"`
	// Maximum number of idle connections in the pool
	MaxIdleConns int `mapstructure:"max_idle_conns"`
}
