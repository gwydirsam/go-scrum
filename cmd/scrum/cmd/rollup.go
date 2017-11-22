package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rollupCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Roll up scrums",
	Long:  `Roll up scrum status for posting in jabber`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return CheckRequiredFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// setup account
		account := "swg"

		mantaURL := viper.GetString("manta_url")
		mantaKeyId := viper.GetString("manta_key_id")
		log.Printf("MANTA_URL: %s\n", mantaURL)
		log.Printf("MANTA_KEY_ID: %s\n", mantaKeyId)

		sshKeySigner, err := authentication.NewSSHAgentSigner(
			mantaKeyId, account)
		if err != nil {
			return errors.Wrap(err, "unable to create new SSH agent signer")
		}

		client, err := manta.NewClient(&manta.ClientOptions{
			Endpoint:    mantaURL,
			AccountName: account,
			Signers:     []authentication.Signer{sshKeySigner},
		})
		if err != nil {
			return errors.Wrap(err, "unable to create a new manta client")
		}

		userName, err := getUser()
		if err != nil {
			return errors.Wrap(err, "unable to get a username")
		}

		// setup time format string to get current date
		layout := "2006/01/02"
		output, err := client.GetObject(&manta.GetObjectInput{
			ObjectPath: path.Join("scrum", time.Now().Format(layout), userName),
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
	RootCmd.AddCommand(rollupCmd)
}
