package cmd

import (
	"context"
	"io"
	"os"
	"path"
	"strings"
	"time"

	tritonError "github.com/joyent/triton-go/errors"
	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setCmd = &cobra.Command{
	Use:          "set",
	SuggestFor:   []string{"push", "put"},
	Short:        "Set scrum information",
	Long:         `Set scrum information, either for yourself (or teammates)`,
	SilenceUsage: true,
	Example: `  $ scrum set -i today.md                         # Set my scrum using today.md
  $ scrum set -u other.username -t -i tomorrow.md # Set other.username's scrum for tomorrow`,
	Args: cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRequiredFlags(cmd.Flags()); err != nil {
			return errors.Wrap(err, "required flag missing")
		}

		if viper.GetBool(configKeySetTomorrow) && viper.GetBool(configKeySetYesterday) {
			return errors.New("tomorrow and yesterday are conflicting optoins")
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
		c, err := getScrumClient()
		if err != nil {
			return errors.Wrap(err, "unable to create a new scrum client")
		}
		defer c.dumpMantaClientStats()

		numDays := viper.GetInt(configKeySetNumDays)
		if numDays < 1 {
			numDays = 1
		}

		numSick := viper.GetInt(configKeySetSickDays)
		numVacation := viper.GetInt(configKeySetVacationDays)

		// Build file string
		inputScrumDate, err := time.Parse(dateInputFormat, viper.GetString(configKeySetInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to parse date")
		}

		switch {
		case viper.GetBool(configKeySetTomorrow):
			inputScrumDate = getTomorrow(inputScrumDate)
		case viper.GetBool(configKeySetYesterday):
			inputScrumDate = getYesterday(inputScrumDate)
		}

		// create end date string for vacation and sick time
		endDate := inputScrumDate
		if numSick > 0 || numVacation > 0 {
			endDate = endDate.AddDate(0, 0, max(numSick, numVacation))
		}

		var foundError bool
	DAY_HANDLING:
		for i := 0; i < numDays; i++ {
			scrumDate := inputScrumDate.AddDate(0, 0, i)

			scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), getUser(configKeySetUsername))

			// Check if scrum exists
			ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
			start := time.Now().UnixNano()
			_, err = c.Objects().Get(ctx, &storage.GetObjectInput{
				ObjectPath: scrumPath,
			})
			elapsed := time.Now().UnixNano() - start
			c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
			c.getCalls++

		ERROR_HANDLING:
			switch {
			case err != nil && tritonError.IsResourceNotFoundError(err):
				// User data doesn't exist
				break ERROR_HANDLING
			case err != nil && !tritonError.IsResourceNotFoundError(err):
				return errors.Wrap(err, "unknown error")
			case err == nil:
				// User data does exist

				// The semantic of "setting a DELETE" operation bugs me, but I don't
				// want to expose a top-level command to delete my user file.
				if viper.GetBool(configKeySetUnlinkDay) {
					if err := unlinkObject(c, scrumPath); err != nil {
						return errors.Wrap(err, "unable to unlink object")
					}

					continue DAY_HANDLING
				}

				if viper.GetBool(configKeySetForce) {
					// If we're overriding multiple days, increase the verbosity of the
					// log messages (vs the common case, overriding just today, in which
					// case we just use the DEBUG level).
					if numDays > 1 {
						log.Info().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("replacing scrum")
					} else {
						log.Debug().Str("path", scrumPath).Bool("force", viper.GetBool(configKeySetForce)).Msg("replacing scrum")
					}

					break ERROR_HANDLING
				} else {
					if numDays == 1 {
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
				}
			}

			var reader io.Reader
			switch {
			case numSick != 0:
				reader = strings.NewReader("Sick leave until " + endDate.Format(scrumDateLayout) + "\n")
			case numVacation != 0:
				reader = strings.NewReader("Vacation until " + endDate.Format(scrumDateLayout) + "\n")
			case viper.GetString(configKeySetFilename) == "":
				return errors.New("empty filename specified, use '-' as the input filename to use stdin")
			case viper.GetString(configKeySetFilename) == "-":
				reader = os.Stdin
			case viper.GetString(configKeySetFilename) != "":
				f, err := os.Open(viper.GetString(configKeySetFilename))
				if err != nil {
					return errors.Wrap(err, "unable to open(2) file")
				}
				defer f.Close()

				sb, err := f.Stat()
				if err != nil {
					return errors.Wrap(err, "unable to stat(2) file")
				}

				if !sb.Mode().IsRegular() {
					return errors.Wrapf(err, "%q is not a regular file", viper.GetString(configKeySetFilename))
				}

				if sb.Size() == 0 {
					return errors.Errorf("scrum input is not a zero-byte file (%q)", viper.GetString(configKeySetFilename))
				}

				reader = f
			}

			if err := putObject(c, scrumPath, reader); err != nil {
				return errors.Wrapf(err, "unable to put object: %q", scrumPath)
			}
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
			key               = configKeySetTomorrow
			longOpt, shortOpt = "tomorrow", "t"
			defaultValue      = false
		)
		setCmd.Flags().BoolP(longOpt, shortOpt, defaultValue, "Set scrum for the next day")
		viper.BindPFlag(key, setCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeySetUsername
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

	{
		const (
			key               = configKeySetYesterday
			longOpt, shortOpt = "yesterday", "y"
			defaultValue      = false
		)
		setCmd.Flags().BoolP(longOpt, shortOpt, defaultValue, "Set scrum for yesterday")
		viper.BindPFlag(key, setCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)

		// I don't want to generally be advertising that this is available.
		setCmd.Flags().MarkHidden(longOpt)
	}

	{
		const (
			key          = configKeySetUnlinkDay
			longOpt      = "rm"
			defaultValue = false
		)
		setCmd.Flags().Bool(longOpt, defaultValue, "Remove scrum for a given day")
		viper.BindPFlag(key, setCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)

		// I don't want to generally be advertising that this is available.
		setCmd.Flags().MarkHidden(longOpt)
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

func putObject(c *scrumClient, scrumPath string, reader io.Reader) error {
	putInput := &storage.PutObjectInput{
		ObjectPath:   scrumPath,
		ObjectReader: reader,
		ForceInsert:  true,
	}

	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
	defer func(start int64) {
		elapsed := time.Now().UnixNano() - start
		c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
		c.putCalls++
	}(time.Now().UnixNano())
	if err := c.Objects().Put(ctx, putInput); err != nil {
		return errors.Wrap(err, "unable to put object")
	}

	log.Info().Str("path", scrumPath).Msg("scrummed")

	return nil
}

func unlinkObject(c *scrumClient, scrumPath string) error {
	deleteInput := &storage.DeleteObjectInput{
		ObjectPath: scrumPath,
	}

	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
	start := time.Now().UnixNano()
	if err := c.Objects().Delete(ctx, deleteInput); err != nil {
		return errors.Wrap(err, "unable to delete object")
	}
	elapsed := time.Now().UnixNano() - start
	c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
	c.deleteCalls++

	log.Info().Str("path", scrumPath).Msg("removed scrum file")

	return nil
}
