package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	content, err := os.ReadFile(userHomeDir + "/.gatorconfig.json")
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) SetUser(user string) error {
	c.CurrentUserName = user
	jsonData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return nil
}
