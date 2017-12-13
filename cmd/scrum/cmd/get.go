package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
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

		color.NoColor = !viper.GetBool(configKeyLogTermColor)

		switch {
		case viper.GetBool(configKeyGetOptAll):
			return getAllScrum(client, scrumDate)
		case !viper.GetBool(configKeyGetOptAll):
			return getSingleScrum(os.Stdout, client, scrumDate, getUser(configKeyGetUsername), false)
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

	const defaultTerminalWidth = 80
	terminalWidth, _, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Warn().Err(err).Msg("unable to get terminal size, using default")
		terminalWidth = defaultTerminalWidth
	}

	horizontalSeparator := strings.Repeat("-", terminalWidth) + "\n"

	var firstError error
	for _, ent := range dirEnts.Entries {
		if v, found := usernameActionMap[ent.Name]; found && v == _Ignore {
			continue
		}

		w.WriteString(horizontalSeparator)

		if err := getSingleScrum(w, c, scrumDate, ent.Name, true); err != nil {
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

func getSingleScrum(w io.Writer, c *storage.StorageClient, scrumDate time.Time, user string, includeHeader bool) error {
	objectPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), user)

	obj, err := c.Objects().Get(context.Background(), &storage.GetObjectInput{
		ObjectPath: objectPath,
	})
	if err != nil {
		return errors.Wrap(err, "unable to get manta object")
	}
	defer obj.ObjectReader.Close()

	body, err := ioutil.ReadAll(obj.ObjectReader)
	if err != nil {
		return errors.Wrap(err, "unable to read manta object")
	}

	if includeHeader {
		key := color.New(color.Bold, color.FgWhite).SprintFunc()
		value := color.New(color.FgWhite, color.Underline).SprintFunc()

		output := []string{
			fmt.Sprintf("%s | %s", key("user"), value(user)),
			fmt.Sprintf("%s | %s", key("mtime"), value(obj.LastModified.Local().String())),
		}
		w.Write([]byte(columnize.SimpleFormat(output) + "\n\n"))
	}

	w.Write(body)

	return nil
}
