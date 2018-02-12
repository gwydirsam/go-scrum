package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/joyent/triton-go/storage"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	{
		const (
			key          = configKeyListUsersOne
			longName     = "usernames"
			shortName    = "1"
			defaultValue = false
			description  = "List usernames only"
		)

		flags := listCmd.Flags()
		flags.BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, flags.Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeyListUsersAll
			longName     = "all"
			shortName    = "a"
			defaultValue = true
			description  = "List all metadata details"
		)

		flags := listCmd.Flags()
		flags.BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, flags.Lookup(longName))
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

		flags := listCmd.Flags()
		flags.StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, flags.Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyListTomorrow
			longOpt, shortOpt = "tomorrow", "t"
			defaultValue      = false
		)
		flags := listCmd.Flags()
		flags.BoolP(longOpt, shortOpt, defaultValue, "List scrums for the next weekday")
		viper.BindPFlag(key, flags.Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyListYesterday
			longOpt, shortOpt = "yesterday", "y"
			defaultValue      = false
		)
		flags := listCmd.Flags()
		flags.BoolP(longOpt, shortOpt, defaultValue, "List scrum for the previous weekday")
		viper.BindPFlag(key, flags.Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Args:         cobra.NoArgs,
	Use:          "list",
	SuggestFor:   []string{"ls", "dir"},
	Short:        "List scrum information",
	Long:         `List scrum information for the day`,
	SilenceUsage: true,
	Example: `  $ scrum list                      # List scrummers for the day
  $ scrum list -t`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRequiredFlags(cmd.Flags()); err != nil {
			return errors.Wrap(err, "required flag missing")
		}

		if viper.GetBool(configKeyListTomorrow) && viper.GetBool(configKeyListYesterday) {
			return errors.New("tomorrow and yesterday are conflicting optoins")
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getScrumClient()
		if err != nil {
			return errors.Wrap(err, "unable to create a new scrum client")
		}
		defer client.dumpMantaClientStats()

		scrumDate, err := getDateInLocation(viper.GetString(configKeyListInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to parse scrum date")
		}

		switch {
		case viper.GetBool(configKeyListTomorrow):
			scrumDate = getNextWeekday(scrumDate)
		case viper.GetBool(configKeyListYesterday):
			scrumDate = getPreviousWeekday(scrumDate)
		}

		return listScrummers(client, scrumDate)
	},
}

// listScrummers prints every user who scrummed
func listScrummers(c *scrumClient, scrumDate time.Time) error {
	scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout))

	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
	start := time.Now()
	dirEnts, err := c.Dir().List(ctx, &storage.ListDirectoryInput{
		DirectoryName: scrumPath,
	})
	elapsed := time.Now().Sub(start)
	log.Debug().Str("path", scrumPath).Str("duration", elapsed.String()).Msg("ListDirectory")
	c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
	c.listCalls++
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
		if viper.GetBool(configKeyUseUTC) {
			tz, _ = scrumDate.UTC().Zone()
		} else {
			tz, _ = scrumDate.Local().Zone()
		}

		table := tablewriter.NewWriter(w)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderLine(false)
		table.SetAutoFormatHeaders(true)

		table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")

		table.SetHeader([]string{"name", "size", fmt.Sprintf("mtime (%s)", tz)})
		if viper.GetBool(configKeyLogTermColor) {
			table.SetHeaderColor(
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
			)
		}

		const mtimeFormat = "2006-01-02 15:04:05"

		var numScrum uint
		for _, ent := range dirEnts.Entries {
			if v, found := usernameActionMap[ent.Name]; found && v == _Ignore {
				continue
			}

			mtime := ent.ModifiedTime
			if !viper.GetBool(configKeyUseUTC) {
				mtime = mtime.Local()
			}

			table.Append([]string{ent.Name, fmt.Sprintf("%d", ent.Size), mtime.Format(mtimeFormat)})
			numScrum++
		}
		table.SetFooter([]string{"Total", fmt.Sprintf("%d", numScrum), ""})

		table.Render()

		return nil
	default:
		return errors.New("unsupported mode of operation")
	}
}
