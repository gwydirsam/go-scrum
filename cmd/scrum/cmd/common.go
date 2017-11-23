package cmd

import (
	"os"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func getMantaClient() (*manta.Client, error) {
	mantaAccount := viper.GetString(configKeyMantaAccount)
	mantaURL := viper.GetString(configKeyMantaURL)
	mantaKeyID := viper.GetString(configKeyMantaKeyID)

	sshKeySigner, err := authentication.NewSSHAgentSigner(
		mantaKeyID, mantaAccount)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create new SSH agent signer")
	}

	client, err := manta.NewClient(&manta.ClientOptions{
		Endpoint:    mantaURL,
		AccountName: mantaAccount,
		Signers:     []authentication.Signer{sshKeySigner},
		Logger:      stdLogger,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a new manta client")
	}

	return client, nil
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
