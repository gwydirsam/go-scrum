package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	{
		const (
			key          = configKeyListUsersOne
			longName     = "list-users-one"
			shortName    = "1"
			defaultValue = false
			description  = "List no metadata"
		)

		listCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, listCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeyListUsersAll
			longName     = "list-users-all"
			shortName    = "a"
			defaultValue = true
			description  = "List all metadata details"
		)

		listCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, listCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key         = configKeyListInputDate
			longName    = "date"
			shortName   = "D"
			description = "Date for scrum"
		)
		defaultValue := time.Now().Format(dateInputFormat)

		listCmd.Flags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, listCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeyListUsers
			longName     = "list-users"
			shortName    = "L"
			defaultValue = false
			description  = "List usernames who scrummed for a given day"
		)

		listCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, listCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyTomorrow
			longOpt, shortOpt = key, "t"
			defaultValue      = false
		)
		listCmd.Flags().BoolP(longOpt, shortOpt, defaultValue, "List scrums for the next day")
		viper.BindPFlag(key, listCmd.Flags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeyListUsersUTC
			longName     = "mtime-utc"
			shortName    = "Z"
			defaultValue = false
			description  = "List mtime data in UTC"
		)

		listCmd.Flags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, listCmd.Flags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:          "list",
	Aliases:      []string{"ls", "dir"},
	Short:        "List scrum information",
	Long:         `List scrum information for the day`,
	SilenceUsage: true,
	Example: `  $ scrum list                      # List scrummers for the day
  $ scrum list -t`,
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

		scrumDate, err := time.Parse(dateInputFormat, viper.GetString(configKeyListInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to parse date")
		}

		switch {
		case viper.GetBool(configKeyTomorrow):
			scrumDate = scrumDate.AddDate(0, 0, 1)
		}

		return listScrummers(client, scrumDate)
	},
}

// listScrummers prints every user who scrummed
func listScrummers(c *storage.StorageClient, scrumDate time.Time) error {
	scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout))

	dirEnts, err := c.Dir().List(context.Background(), &storage.ListDirectoryInput{
		DirectoryName: scrumPath,
	})
	if err != nil {
		return errors.Wrap(err, "unable to list manta directory")
	}

	if dirEnts.ResultSetSize == 0 {
		log.Warn().Msg("no users have scrummed yet")
		return nil
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	switch {
	case viper.IsSet(configKeyListUsersOne) && viper.GetBool(configKeyListUsersOne):
		for _, ent := range dirEnts.Entries {
			if v, found := usernameActionMap[ent.Name]; found && v == _Ignore {
				continue
			}

			fmt.Fprintln(w, ent.Name)
		}

		return nil
	case viper.GetBool(configKeyListUsersAll):
		var tz string
		if viper.GetBool(configKeyListUsersUTC) {
			tz, _ = scrumDate.UTC().Zone()
		} else {
			tz, _ = scrumDate.Local().Zone()
		}

		output := make([]string, 0, dirEnts.ResultSetSize+1)
		output = append(output, fmt.Sprintf(" name | size | mtime (%s)", tz))

		const mtimeFormat = "2006/01/02 15:04:05"

		for _, ent := range dirEnts.Entries {
			if v, found := usernameActionMap[ent.Name]; found && v == _Ignore {
				continue
			}

			mtime := ent.ModifiedTime
			if !viper.GetBool(configKeyListUsersUTC) {
				mtime = mtime.Local()
			}

			l := fmt.Sprintf("%s | %d | %s", ent.Name, ent.Size, mtime.Format(mtimeFormat))
			output = append(output, l)
		}

		fmt.Fprintln(w, columnize.SimpleFormat(output))

		return nil
	default:
		return errors.New("unsupported mode of operation")
	}
}
