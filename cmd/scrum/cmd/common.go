package cmd

import (
	"os"

	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func getMantaClient() (*storage.StorageClient, error) {
	mantaAccount := viper.GetString(configKeyMantaAccount)
	mantaURL := viper.GetString(configKeyMantaURL)
	mantaKeyID := viper.GetString(configKeyMantaKeyID)

	sshKeySigner, err := authentication.NewSSHAgentSigner(
		mantaKeyID, mantaAccount)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create new SSH agent signer")
	}

	c, err := storage.NewClient(&triton.ClientConfig{
		MantaURL:    mantaURL,
		AccountName: mantaAccount,
		Signers:     []authentication.Signer{sshKeySigner},
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a new manta client")
	}

	return c, nil
}

func getUser() string {
	switch {
	case viper.IsSet(configKeyUsername):
		return interpolateValue(viper.GetString(configKeyUsername))
	case viper.IsSet(configKeyMantaUser):
		return interpolateValue(viper.GetString(configKeyMantaUser))
	}

	user := viper.GetString(configKeyMantaUser)
	if user != "" {
		return interpolateValue(user)
	}

	user = viper.GetString(configKeyUsername)
	if user != "" {
		return interpolateValue(user)
	}

	log.Warn().Msgf("unable to detect a username, please set %q or %q in %q", configKeyUsername, configKeyMantaUser, viper.ConfigFileUsed)
	return ""
}

func interpolateValue(val string) string {
	switch val {
	case "$MANTA_USER":
		return os.Getenv("MANTA_USER")
	case "$USER":
		return os.Getenv("USER")
	}

	return val
}
