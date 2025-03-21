package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ray-d-song/goread/pkg/ui"
)

// State represents the reading state of a file
type State struct {
	Index       int            `json:"index"`
	Width       int            `json:"width"`
	ColorScheme ui.ColorScheme `json:"color_scheme"`
	Pos         int            `json:"pos"`
	Pctg        float64        `json:"pctg"`
	LastRead    bool           `json:"lastread"`
}

// Config represents the configuration of the application
type Config struct {
	States     map[string]State `json:"states"`
	ConfigFile string           `json:"-"`
}

// NewConfig creates a new Config instance
func NewConfig() (*Config, error) {
	configFile, err := getConfigFile()
	if err != nil {
		return nil, err
	}

	config := &Config{
		States:     make(map[string]State),
		ConfigFile: configFile,
	}

	// Load the config if it exists
	if _, err := os.Stat(configFile); err == nil {
		err = config.Load()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Load loads the configuration from the config file
func (c *Config) Load() error {
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.States)
}

// Save saves the configuration to the config file
func (c *Config) Save() error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(c.ConfigFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(c.States, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.ConfigFile, data, 0644)
}

// GetState returns the state of a file
func (c *Config) GetState(file string) (State, bool) {
	state, ok := c.States[file]
	return state, ok
}

// SetState sets the state of a file
func (c *Config) SetState(file string, state State) {
	c.States[file] = state
}

// SetLastRead sets the last read file
func (c *Config) SetLastRead(file string) {
	// Reset all lastread flags
	for f, state := range c.States {
		if state.LastRead {
			state.LastRead = false
			c.States[f] = state
		}
	}

	// Set the lastread flag for the given file
	if state, ok := c.States[file]; ok {
		state.LastRead = true
		c.States[file] = state
	}
}

// GetLastRead returns the last read file
func (c *Config) GetLastRead() (string, bool) {
	for file, state := range c.States {
		if state.LastRead {
			return file, true
		}
	}
	return "", false
}

// getConfigFile returns the path to the config file
func getConfigFile() (string, error) {
	// Try $HOME/.config/goread/config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".config", "goread")
	configFile := filepath.Join(configDir, "config")

	// If the directory exists or can be created, use it
	if _, err := os.Stat(configDir); err == nil || os.IsNotExist(err) {
		return configFile, nil
	}

	// Otherwise, use $HOME/.goread
	return filepath.Join(homeDir, ".goread"), nil
}
