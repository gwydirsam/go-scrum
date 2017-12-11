package cmd

import (
	"context"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/joyent/triton-go/client"
	"github.com/joyent/triton-go/storage"
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
		if err := checkRequiredFlags(cmd.Flags()); err != nil {
			return errors.Wrap(err, "required flag missing")
		}

		return nil
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

		c, err := getMantaClient()
		if err != nil {
			return errors.Wrap(err, "unable to create a new manta client")
		}

		numDays := viper.GetInt(configKeySetNumDays)
		numSick := viper.GetInt(configKeySetSickDays)
		numVacation := viper.GetInt(configKeySetVacationDays)

		// Build file string
		scrumDate, err := time.Parse(dateInputFormat, viper.GetString(configKeySetInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to parse date")
		}

		switch {
		case viper.GetBool(configKeyTomorrow):
			scrumDate = scrumDate.AddDate(0, 0, 1)
		case numDays != 0:
			scrumDate = scrumDate.AddDate(0, 0, numDays)
		}

		// create end date string for vacation and sick time
		endDate := scrumDate
		daysToScrum := 1
		if numSick > 0 || numVacation > 0 {
			daysToScrum = max(numSick, numVacation)
			endDate = endDate.AddDate(0, 0, daysToScrum)
		}

		var foundError bool
	DAY_HANDLING:
		for i := daysToScrum; i > 0; i-- {
			scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), getUser())

			// Check if scrum exists
			_, err = c.Objects().Get(context.TODO(), &storage.GetObjectInput{
				ObjectPath: scrumPath,
			})

		ERROR_HANDLING:
			switch {
			case err != nil && client.IsDirectoryDoesNotExistError(err):
				dirs := strings.Split(scrumDate.Format(scrumDateLayout), "/")
				scrumDir := make([]string, 0, len(dirs)+1)
				scrumDir = append(scrumDir, "scrum")

				// Unconditionally attempt to create all directories in the path
				for _, dir := range dirs {
					scrumDir = append(scrumDir, dir)
					err = c.Dir().Put(context.TODO(), &storage.PutDirectoryInput{
						DirectoryName: path.Join(scrumDir...),
					})
					if err != nil {
						return errors.Wrap(err, "unable to put object")
					}
				}
			case err != nil && !client.IsResourceNotFoundError(err):
				if viper.GetBool(configKeySetForce) {
					// If we're overriding multiple days, increase the verbosity of the
					// log messages (vs the common case, overriding just today, in which
					// case we just use the DEBUG level).
					if daysToScrum > 1 {
						log.Info().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("replacing scrum")
					} else {
						log.Debug().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("replacing scrum")
					}

					break ERROR_HANDLING
				}

				if daysToScrum == 1 {
					log.Error().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("scrum exists, not replacing scrum without -f to override")
					return errors.Wrap(err, "scrum already exists")
				}

				// Let users attempt to stamp out scrum for multiple days and skip over
				// days that already exist.  Return an error just to let the user know
				// that the command did run into a potential problem (i.e. don't return
				// cleanly).
				foundError = true
				log.Info().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("replacing scrum")

				continue DAY_HANDLING
			case err == nil:
				if viper.GetBool(configKeySetForce) {
					log.Debug().Str("path", scrumPath).Msg("scrum already exists, overriding")
					break ERROR_HANDLING
				} else {
					log.Warn().Str("path", scrumPath).Msg("scrum already exists, specify -f to override")
				}
				continue DAY_HANDLING
			}

			var reader io.ReadSeeker
			switch {
			case numSick != 0:
				reader = strings.NewReader("Sick leave until " + endDate.Format(scrumDateLayout) + "\n")
			case numVacation != 0:
				reader = strings.NewReader("Vacation until " + endDate.Format(scrumDateLayout) + "\n")
			case viper.GetString(configKeySetFilename) != "":
				f, err := os.Open(viper.GetString(configKeySetFilename))
				if err != nil {
					return errors.Wrap(err, "unable to open file")
				}
				defer f.Close()
				reader = f
			}

			if err := putObject(c, scrumPath, reader); err != nil {
				return errors.Wrap(err, "unable to put object")
			}

			// scrum for next day
			scrumDate = scrumDate.AddDate(0, 0, 1)
		}

		if foundError {
			return errors.New("error occured while running scrum")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setCmd)

	{
		const (
			key         = configKeySetInputDate
			longName    = "date"
			shortName   = "D"
			description = "Date for scrum"
		)
		defaultValue := time.Now().Format(dateInputFormat)

		setCmd.Flags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

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
			description  = "Recycle scrum update for N days"
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
			description  = "Sick leave for N days"
		)

		setCmd.Flags().UintP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, setCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyTomorrow
			longOpt, shortOpt = key, "t"
			defaultValue      = false
		)
		setCmd.Flags().BoolP(longOpt, shortOpt, defaultValue, "Set scrum for the next day")
		viper.BindPFlag(key, setCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyUsername
			longOpt, shortOpt = "user", "u"
			defaultValue      = "$USER"
		)
		setCmd.Flags().StringP(longOpt, shortOpt, defaultValue, "Set scrum for specified user")
		viper.BindPFlag(key, setCmd.Flags().Lookup(longOpt))
		viper.BindEnv(key, "USER")
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

func putObject(c *storage.StorageClient, scrumPath string, reader io.ReadSeeker) error {
	putInput := &storage.PutObjectInput{
		ObjectPath:   scrumPath,
		ObjectReader: reader,
	}

	if err := c.Objects().Put(context.TODO(), putInput); err != nil {
		return errors.Wrap(err, "unable to put object")
	}

	log.Info().Str("path", scrumPath).Msg("scrummed")

	return nil
}
