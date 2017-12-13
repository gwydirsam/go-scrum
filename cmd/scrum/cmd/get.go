package cmd

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	{
		const (
			key          = configKeyGetOptAll
			longName     = "all"
			shortName    = "a"
			defaultValue = false
			description  = "Get scrum for all users"
		)

		getCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, getCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key         = configKeyGetInputDate
			longName    = "date"
			shortName   = "D"
			description = "Date for scrum"
		)
		defaultValue := time.Now().Format(dateInputFormat)

		getCmd.Flags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, getCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyGetTomorrow
			longOpt, shortOpt = "tomorrow", "t"
			defaultValue      = false
		)
		getCmd.Flags().BoolP(longOpt, shortOpt, defaultValue, "Get scrum for the next day")
		viper.BindPFlag(key, getCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyGetUsername
			longOpt, shortOpt = "user", "u"
			defaultValue      = "$USER"
		)
		getCmd.Flags().StringP(longOpt, shortOpt, defaultValue, "Get scrum for specified user")
		viper.BindPFlag(key, getCmd.Flags().Lookup(longOpt))
		viper.BindEnv(key, "USER")
		viper.SetDefault(key, defaultValue)
	}

	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:          "get",
	Short:        "Get scrum information",
	Long:         `Get scrum information, either for yourself or teammates`,
	SilenceUsage: true,
	Example: `  $ scrum get                      # Get my scrum for today
  $ scrum get -t -u other.username # Get other.username's scrum for tomorrow`,
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

		scrumDate, err := time.Parse(dateInputFormat, viper.GetString(configKeyGetInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to parse date")
		}

		switch {
		case viper.GetBool(configKeyGetTomorrow):
			scrumDate = scrumDate.AddDate(0, 0, 1)
		}

		switch {
		case viper.GetBool(configKeyGetOptAll):
			return getAllScrum(client, scrumDate)
		case !viper.GetBool(configKeyGetOptAll):
			return getSingleScrum(os.Stdout, client, scrumDate, getUser(configKeyGetUsername))
		default:
			return errors.New("unsupported get mode")
		}
	},
}

func getAllScrum(c *storage.StorageClient, scrumDate time.Time) error {
	scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout))

	dirEnts, err := c.Dir().List(context.Background(), &storage.ListDirectoryInput{
		DirectoryName: scrumPath,
	})
	if err != nil {
		return errors.Wrap(err, "unable to list manta directory")
	}

	if dirEnts.ResultSetSize == 0 {
		log.Error().Time("scrum-date", scrumDate).Msg("no users have scrummed for this day")
		return nil
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var firstError error
	for _, ent := range dirEnts.Entries {
		if v, found := usernameActionMap[ent.Name]; found && v == _Ignore {
			continue
		}

		if err := getSingleScrum(w, c, scrumDate, ent.Name); err != nil {
			log.Error().Err(err).Str("username", ent.Name).Msg("unable to get user's scrum")
			if firstError == nil {
				firstError = err
			}
		}
	}

	if firstError != nil {
		return firstError
	}

	return nil
}

func getSingleScrum(w io.Writer, c *storage.StorageClient, scrumDate time.Time, user string) error {
	objectPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), user)

	output, err := c.Objects().Get(context.Background(), &storage.GetObjectInput{
		ObjectPath: objectPath,
	})
	if err != nil {
		return errors.Wrap(err, "unable to get manta object")
	}
	defer output.ObjectReader.Close()

	body, err := ioutil.ReadAll(output.ObjectReader)
	if err != nil {
		return errors.Wrap(err, "unable to read manta object")
	}

	w.Write(body)

	return nil
}
