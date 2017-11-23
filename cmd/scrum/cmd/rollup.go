package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var rollupCmd = &cobra.Command{
	Use:   "rollup",
	Short: "Roll up scrums",
	Long:  `Roll up scrum status for posting in jabber`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRequiredFlags(cmd.Flags()); err != nil {
			return errors.Wrap(err, "required flag missing")
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getMantaClient()
		if err != nil {
			return errors.Wrap(err, "unable to create a new manta client")
		}

		// setup time format string to get current date
		output, err := client.GetObject(&manta.GetObjectInput{
			ObjectPath: path.Join("scrum", time.Now().Format(scrumDateLayout), getUser()),
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
