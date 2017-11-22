package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rollupCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Roll up scrums",
	Long:  `Roll up scrum status for posting in jabber`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return checkRequiredFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// setup account
		account := "swg"

		mantaURL := viper.GetString("manta_url")
		mantaKeyID := viper.GetString("manta_key_id")
		log.Debug().Str("MANTA_URL", mantaURL).Str("MANTA_KEY_ID", mantaKeyID).Msg("")

		sshKeySigner, err := authentication.NewSSHAgentSigner(
			mantaKeyID, account)
		if err != nil {
			return errors.Wrap(err, "unable to create new SSH agent signer")
		}

		client, err := manta.NewClient(&manta.ClientOptions{
			Endpoint:    mantaURL,
			AccountName: account,
			Signers:     []authentication.Signer{sshKeySigner},
			Logger:      stdLogger,
		})
		if err != nil {
			return errors.Wrap(err, "unable to create a new manta client")
		}

		userName, err := getUser()
		if err != nil {
			return errors.Wrap(err, "unable to get a username")
		}

		// setup time format string to get current date
		output, err := client.GetObject(&manta.GetObjectInput{
			ObjectPath: path.Join("scrum", time.Now().Format(scrumDateLayout), userName),
		})
		if err != nil {
			return errors.Wrap(err, "unable to get manta object")
		}

		defer output.ObjectReader.Close()
		body, err := ioutil.ReadAll(output.ObjectReader)
		if err != nil {
			return errors.Wrap(err, "unable to read manta object")
		}

		fmt.Printf("%s", string(body))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rollupCmd)
}
