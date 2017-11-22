package cmd

import (
	"io"
	"os"
	"path"
	"strings"
	"time"

	manta "github.com/jen20/manta-go"
	"github.com/jen20/manta-go/authentication"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setCmd = &cobra.Command{
	Use:          "set",
	Short:        "Set scrum information",
	Long:         `Set scrum information, either for yourself or teammates`,
	SilenceUsage: true,
	Example: `  $ scrum set -i today.md                         # Set my scrum using today.md
  $ scrum set -t -u other.username -i tomorrow.md # Set other.username's scrum for tomorrow`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return checkRequiredFlags(cmd.Flags())
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		//// validate username
		//json, err := ioutil.ReadFile("team.json")
		//if err != nil {
		//	log.Fatalf("ioutil.ReadFile: %s", err)
		//}
		//result := gjson.GetBytes(json, userName)
		//if result.String() == "" {
		//	log.Fatalf("Expected Engineer")
		//}

		// setup account
		account := "Joyent_Dev"
		mantaURL := viper.GetString(configKeyMantaURL)
		mantaKeyID := viper.GetString(configKeyMantaKeyID)

		// setup client
		sshKeySigner, err := authentication.NewSSHAgentSigner(
			mantaKeyID, account)
		if err != nil {
			return errors.Wrap(err, "unable to create a new SSH signer")
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

		numDays := viper.GetInt(configKeySetNumDays)
		numSick := viper.GetInt(configKeySetSickDays)
		numVacation := viper.GetInt(configKeySetVacationDays)

		// Build file string
		// setup time format string to get current date
		scrumDate := time.Now()
		switch {
		case viper.GetBool(configKeyTomorrow):
			scrumDate = scrumDate.AddDate(0, 0, 1)
		case numDays != 0:
			scrumDate = scrumDate.AddDate(0, 0, numDays)
		}

		// create end date string for vacation and sick time
		endDate := time.Now()
		daysToScrum := 1
		if numSick > 0 || numVacation > 0 {
			daysToScrum = max(numSick, numVacation)
			endDate = endDate.AddDate(0, 0, daysToScrum)
		}

		userName, err := getUser()
		if err != nil {
			return errors.Wrap(err, "unable to get username")
		}

		for i := daysToScrum; i > 0; i-- {
			scrumPath := path.Join("scrum", scrumDate.Format(scrumDateLayout), userName)

			// Check if scrum exists
			_, err = client.GetObject(&manta.GetObjectInput{
				ObjectPath: scrumPath,
			})

			switch {
			case err != nil && manta.IsDirectoryDoesNotExistError(err):
				dirs := strings.Split(scrumDate.Format(scrumDateLayout), "/")
				scrumDir := make([]string, 0, len(dirs)+1)
				scrumDir = append(scrumDir, "scrum")
				for _, dir := range dirs {
					scrumDir = append(scrumDir, dir)
					err = client.PutDirectory(&manta.PutDirectoryInput{
						DirectoryName: path.Join(scrumDir...),
					})
					if err != nil {
						return errors.Wrap(err, "unable to put object")
					}
				}
			case err != nil && !manta.IsResourceNotFoundError(err):
				return errors.Wrap(err, "unable to get object")
			case !viper.GetBool(configKeySetForce):
				// if not, we need a force flag
				return errors.Wrapf(err, "~~/stor/%s exists and -f not specified", scrumPath)
			case err == nil:
				if !viper.GetBool(configKeySetForce) {
					log.Warn().Str("path", scrumPath).Msg("scrum already exists, specify -f to override")
				}
				continue
			}

			var reader io.ReadSeeker
			if numSick != 0 {
				reader = strings.NewReader("Sick leave until " + endDate.Format(scrumDateLayout) + "\n")
			} else if numVacation != 0 {
				reader = strings.NewReader("Vacation until " + endDate.Format(scrumDateLayout) + "\n")
			} else if viper.GetString(configKeySetFilename) != "" {
				f, err := os.Open(viper.GetString(configKeySetFilename))
				if err != nil {
					return errors.Wrap(err, "unable to open file")
				}
				defer f.Close()
				reader = f
			}

			if err := putObject(client, scrumPath, reader); err != nil {
				return errors.Wrap(err, "unable to put object")
			}

			// scrum for next day
			scrumDate = scrumDate.AddDate(0, 0, 1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setCmd)

	{
		const (
			key          = configKeySetForce
			longName     = "force"
			shortName    = "f"
			defaultValue = false
			description  = "Force overwrite of any present scrum"
		)

		setCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeySetNumDays
			longName     = "days"
			shortName    = "d"
			defaultValue = 0
			description  = "Scrum for n days from now"
		)

		setCmd.Flags().UintP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeySetSickDays
			longName     = "sick"
			shortName    = "s"
			defaultValue = 0
			description  = "Sick leave for n days"
		)

		setCmd.Flags().UintP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeySetVacationDays
			longName     = "vacation"
			shortName    = "v"
			defaultValue = 0
			description  = "Vacation for N days"
		)

		setCmd.Flags().UintP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeySetFilename
			longName     = "file"
			shortName    = "i"
			defaultValue = ""
			description  = "File to read scrum from"
		)

		setCmd.Flags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	// Required Flags
	setCmd.MarkFlagRequired(configKeyMantaUser)
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func putObject(client *manta.Client, scrumPath string, reader io.ReadSeeker) error {
	err := client.PutObject(&manta.PutObjectInput{
		ObjectPath:   scrumPath,
		ObjectReader: reader,
	})
	if err != nil {
		return errors.Wrap(err, "unable to put object")
	}
	log.Info().Str("path", scrumPath).Msg("scrummed")

	return nil
}
