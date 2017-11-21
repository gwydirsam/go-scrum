package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
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
	Run: func(cmd *cobra.Command, args []string) {
		// setup account
		account := "swg"

		mantaURL = viper.GetString("manta_url")
		mantaKeyId = viper.GetString("manta_key_id")
		log.Printf("MANTA_URL: %s\n", mantaURL)
		log.Printf("MANTA_KEY_ID: %s\n", mantaKeyId)

		sshKeySigner, err := authentication.NewSSHAgentSigner(
			mantaKeyId, account)
		if err != nil {
			log.Fatalf("NewSSHAgentSigner: %s", err)
		}

		client, err := manta.NewClient(&manta.ClientOptions{
			Endpoint:    mantaURL,
			AccountName: account,
			Signers:     []authentication.Signer{sshKeySigner},
		})
		if err != nil {
			log.Fatalf("NewClient: %s", err)
		}

		// setup time format string to get current date
		layout := "2006/01/02"
		output, err := client.GetObject(&manta.GetObjectInput{
			ObjectPath: "scrum/" + time.Now().Format(layout) + "/" + userName,
		})
		if err != nil {
			log.Fatalf("GetObject(): %s", err)
		}

		defer output.ObjectReader.Close()
		body, err := ioutil.ReadAll(output.ObjectReader)
		if err != nil {
			log.Fatalf("Reading Object: %s", err)
		}

		fmt.Printf("%s", string(body))
	},
}

func init() {
	RootCmd.AddCommand(rollupCmd)
}
