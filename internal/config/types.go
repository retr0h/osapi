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
	API
	Task
	// Debug enable or disable debug option set from CLI.
	Debug bool `mapstruture:"debug"`
}

// API configuration settings.
type API struct {
	Client
	Server
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

// Task configuration settings.
type Task struct {
	Server TaskServer `mapstructure:"server,omitempty"`
}

// TaskServer configuration settings.
type TaskServer struct {
	// Host bind the server to localhost.
	Host string `mapstructure:"host"`
	// Port the server will bind to.
	Port int `mapstructure:"port"`
	// Trace enable detailed tracing for debugging.
	Trace bool `mapstructure:"trace"`
	// Debug enable debug-level logging.
	Debug bool `mapstructure:"debug"`
	// NoLog enable logging for server events.
	NoLog bool `mapstructure:"no_log"`
	// FileStoreDir JetStream data will be persisted here.
	FileStoreDir string `mapstructure:"file_store_dir"`
}
