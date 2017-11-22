package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var getCmd = &cobra.Command{
	Use:          "get",
	Short:        "Get scrum information",
	Long:         `Get scrum information, either for yourself or teammates`,
	SilenceUsage: true,
	Example: `  $ scrum get                      # Get my scrum for today
  $ scrum get -t -u other.username # Get other.username's scrum for tomorrow`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return checkRequiredFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// setup account
		account := "Joyent_Dev"

		mantaURL := viper.GetString(configKeyMantaURL)
		mantaKeyID := viper.GetString(configKeyMantaKeyID)

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
			return errors.Wrap(err, "unable to find a username")
		}

		// setup time format string to get current date
		scrumDate := time.Now()
		switch {
		case viper.GetBool(configKeyTomorrow):
			scrumDate = scrumDate.AddDate(0, 0, 1)
		}

		output, err := client.GetObject(&manta.GetObjectInput{
			ObjectPath: path.Join("scrum", scrumDate.Format(scrumDateLayout), userName),
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
	rootCmd.AddCommand(getCmd)
	getCmd.MarkFlagRequired("user")
}
