package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func getUser() (string, error) {
	username := viper.GetString("user")
	if username == "$USER" {
		user, found := os.LookupEnv("USER")
		if !found {
			return "", fmt.Errorf("%s requires a flag (or the MANTA_USER or USER environment variable)", "user")
		}

		return user, nil
	}

	if username == "" {
		return "", fmt.Errorf("%s can not be an empty string", "user")
	}

	return username, nil
}
