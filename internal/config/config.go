// Package config
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getFilePath() (string, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	fullPath := fmt.Sprintf("%s/.gatorconfig.json", homePath)
	return fullPath, nil
}

func Read() (Config, error) {
	fullPath, err := getFilePath()
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

func write(c Config) error {
	fullPath, err := getFilePath()
	if err != nil {
		return err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = file.Write(jsonBytes)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	err := write(*c)
	if err != nil {
		return err
	}
	return nil
}
