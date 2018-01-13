package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/gwydirsam/go-scrum/cmd/scrum/buildtime"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:          "init",
	Short:        "Generate an initial scrum configuration file",
	Long:         `Generate an initial scrum configuraiton file`,
	SilenceUsage: true,
	Example: `  $ scrum init                 # Create a new scrum config file
  $ scrum init -f -            # Write the config file to stdout
  $ scrum init -f ./scrum.toml # Create a new scrum config file`,
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		var b bytes.Buffer
		b.WriteString("[general]\n")
		b.WriteString(fmt.Sprintf("#country = %+q\n", viper.GetString(configKeyCountry)))
		b.WriteString("\n")
		b.WriteString("[highlight]\n")
		b.WriteString(fmt.Sprintf("#keyword   = %+q # exact match \"keyword\"\n", "red underline"))
		b.WriteString(fmt.Sprintf("#\"substr~\" = %+q  # substring match \"substr\"\n", "italic green"))
		b.WriteString(fmt.Sprintf("#\"fuzzy~2\" = %+q  # match \"fuzzy\" with a distance of 2\n", "reverse blue"))
		b.WriteString("\n")
		b.WriteString("[log]\n")
		b.WriteString(fmt.Sprintf("#format    = %+q\n", viper.GetString(configKeyLogFormat)))
		b.WriteString(fmt.Sprintf("#level     = %+q\n", viper.GetString(configKeyLogLevel)))
		b.WriteString(fmt.Sprintf("#stats     = %t\n", viper.GetBool(configKeyLogStats)))
		b.WriteString(fmt.Sprintf("#use-color = %t\n", viper.GetBool(configKeyLogTermColor)))
		b.WriteString("\n")
		b.WriteString("[manta]\n")
		b.WriteString(fmt.Sprintf("#account = %+q\n", viper.GetString(configKeyMantaAccount)))

		// TODO(seanc@): Pull this value out of the following in order to reduce the
		// fiction associated with using Manta.
		//
		//   ssh-keygen -E md5 -lf ~/.ssh/id_rsa.pub | awk '{print $2}' | cut -d : -f 2-
		b.WriteString(fmt.Sprintf("#key-id  = %+q\n", viper.GetString(configKeyMantaKeyID)))
		b.WriteString(fmt.Sprintf("#timeout = %+q\n", viper.GetDuration(configKeyMantaTimeout)))
		b.WriteString(fmt.Sprintf("#url     = %+q\n", viper.GetString(configKeyMantaURL)))
		b.WriteString(fmt.Sprintf("#user    = %+q\n", viper.GetString(configKeyMantaUser)))

		rawFilename := viper.GetString(configKeyInitFilename)
		if rawFilename == "-" {
			b.WriteTo(os.Stdout)
			return nil
		}

		filename, err := homedir.Expand(rawFilename)
		if err != nil {
			return errors.Wrap(err, "unable to find a user's home directory")
		}

		if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
			return errors.Wrap(err, "unable to create config directory")
		}

		f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Wrap(err, "unable to create config file")
		}

		n, err := b.WriteTo(f)
		if err != nil {
			return errors.Wrap(err, "unable to write config file")
		}

		if err := f.Close(); err != nil {
			return errors.Wrap(err, "unable to close file")
		}
		log.Info().Int64("bytes-written", n).Str("filename", filename).Msg("wrote config file")

		return nil
	},
}

func init() {
	{
		const (
			key         = configKeyInitFilename
			longName    = "file"
			shortName   = "f"
			description = "Config file to initialize"
		)
		defaultValue := path.Join("~/", ".config", buildtime.PROGNAME, buildtime.PROGNAME+".toml")

		initCmd.Flags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, initCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	rootCmd.AddCommand(initCmd)
}
