package cmd

import (
	"os"
	"time"

	"github.com/circonus-labs/circonusllhist"
	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// scrumClient wraps a StorateClient and a Histogram
type scrumClient struct {
	*storage.StorageClient

	// Time per operation (us)
	*circonusllhist.Histogram

	// Count of each operation type
	deleteCalls uint64
	getCalls    uint64
	listCalls   uint64
	putCalls    uint64
}

func (sc *scrumClient) dumpMantaClientStats() {
	if !viper.GetBool(configKeyLogStats) {
		return
	}

	log.Info().
		Uint64("delete-calls", sc.deleteCalls).
		Uint64("get-calls", sc.getCalls).
		Uint64("list-calls", sc.listCalls).
		Uint64("put-calls", sc.putCalls).
		Str("max", (time.Duration(sc.Histogram.Max()*float64(time.Second))).String()).
		Str("min", (time.Duration(sc.Histogram.Min()*float64(time.Second))).String()).
		Str("mean", (time.Duration(sc.Histogram.Mean()*float64(time.Second))).String()).
		Str("tp90", (time.Duration(sc.Histogram.ValueAtQuantile(0.90)*float64(time.Second))).String()).
		Str("tp95", (time.Duration(sc.Histogram.ValueAtQuantile(0.95)*float64(time.Second))).String()).
		Str("tp99", (time.Duration(sc.Histogram.ValueAtQuantile(0.99) * float64(time.Second))).String()).
		Msg("stats")
}

func getScrumClient() (*scrumClient, error) {
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

	tsc, err := storage.NewClient(&triton.ClientConfig{
		MantaURL:    mantaURL,
		AccountName: mantaAccount,
		Signers:     []authentication.Signer{sshKeySigner},
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a new manta client")
	}

	return &scrumClient{
		StorageClient: tsc,
		Histogram:     circonusllhist.New(),
	}, nil
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
