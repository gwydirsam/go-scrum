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

	input := authentication.SSHAgentSignerInput{
		KeyID:       viper.GetString(configKeyMantaKeyID),
		AccountName: mantaAccount,
	}
	sshKeySigner, err := authentication.NewSSHAgentSigner(input)
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

func getUser(userKey string) string {
	switch {
	case viper.IsSet(userKey):
		return interpolateValue(viper.GetString(userKey))
	case viper.IsSet(configKeyMantaUser):
		return interpolateValue(viper.GetString(configKeyMantaUser))
	}

	user := viper.GetString(configKeyMantaUser)
	if user != "" {
		return interpolateValue(user)
	}

	user = viper.GetString(userKey)
	if user != "" {
		return interpolateValue(user)
	}

	log.Warn().Msgf("unable to detect a username, please set %q or %q in %q", userKey, configKeyMantaUser, viper.ConfigFileUsed)
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
