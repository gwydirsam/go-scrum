package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
			key               = configKeyGetTomorrow
			longOpt, shortOpt = "tomorrow", "t"
			defaultValue      = false
		)
		flags := editCmd.Flags()
		flags.BoolP(longOpt, shortOpt, defaultValue, "Get scrum for the next weekday")
		viper.BindPFlag(key, flags.Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyGetYesterday
			longOpt, shortOpt = "yesterday", "y"
			defaultValue      = false
		)
		flags := editCmd.Flags()
		flags.BoolP(longOpt, shortOpt, defaultValue, "Get scrum for the previous weekday")
		viper.BindPFlag(key, flags.Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:          "edit",
	Short:        "Edit scrum information",
	Long:         `Edit scrum information, either for yourself (or teammates)`,
	SilenceUsage: true,
	Example:      `  $ scrum edit -t          # Edit my scrum for tomorrow`,
	Args:         cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkRequiredFlags(cmd.Flags()); err != nil {
			return errors.Wrap(err, "required flag missing")
		}

		if viper.GetBool(configKeyGetTomorrow) && viper.GetBool(configKeyGetYesterday) {
			return errors.New("tomorrow and yesterday are conflicting optoins")
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getScrumClient()
		if err != nil {
			return errors.Wrap(err, "unable to create a new manta client")
		}
		defer client.dumpMantaClientStats()

		scrumDate, err := getDateInLocation(viper.GetString(configKeyGetInputDate))
		if err != nil {
			return errors.Wrap(err, "unable to get scrum date")
		}

		switch {
		case viper.GetBool(configKeyGetTomorrow):
			scrumDate = getNextWeekday(scrumDate)
		case viper.GetBool(configKeyGetYesterday):
			scrumDate = getPreviousWeekday(scrumDate)
		}

		username := viper.GetString(configKeyScrumUsername)
		username = interpolateUserEnvVar(username)

		// 1a) if before 9:27am Pacific or force flag set, edit today's scrum
		// 1b) if no scrum file is set for today, copy yesterday's scrum
		// 1c) if no scrum file is set for yesterday or today, create a new scrum
		//     from a template
		// 1d) if after 9:27am Pacific and no force flag is set, edit tomorrow's
		//     scrum using today's scrum file

		// 2) edit scrum
		// 3) post scrum

		dir, err := ioutil.TempDir("", "scrum")
		if err != nil {
			return errors.Wrap(err, "unable to create tempdir")
		}
		defer os.RemoveAll(dir)

		tmpFilename := filepath.Join(dir, fmt.Sprintf("%s-%s.md", username, scrumDate.Format("2006-01-02")))
		switch {
		case !fileExists,
			fileExists && viper.GetBool(configKeyEditForce):
			if err := fetchScrum(client, scrumDate, username, tmpFilename, viper.GetBool(configKeyEditForce)); err != nil {
				return errors.Wrap(err, "unable to fetch scrum during edit")
			}
		}

		if err := editScrum(tmpFilename); err != nil {
			return errors.Wrap(err, "unable to edit scrum")
		}

		scrumPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), username)
		if err := putObject(client, scrumPath, r); err != nil {
			return errors.Wrap(err, "unable to put scrum during edit")
		}

	},
}

func fetchScrum(c *scrumClient, scrumDate time.Time, username, filename string, force bool) error {
	openFlags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	f, err := os.OpenFile(filename, openFlags, 0644)
	if err != nil {
		return errors.Wrap(err, "unable to open tmp file")
	}
	var closed bool
	defer func() {
		if !closed {
			if err := f.Close(); err != nil {
				log.Warn().Err(err).Str("filename", filename).Msg("unable to close file")
			}
		}
	}()

	objectPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), user)

	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
	start := time.Now()
	obj, err := c.Objects().Get(ctx, &storage.GetObjectInput{
		ObjectPath: objectPath,
	})
	elapsed := time.Now().Sub(start)
	log.Debug().Str("path", objectPath).Str("duration", elapsed.String()).Msg("GetObject")
	c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
	c.getCalls++
	if err != nil {
		return errors.Wrap(err, "unable to get manta object")
	}
	defer obj.ObjectReader.Close()

	fBuf := bufio.NewWriter(f)
	rBuf := bufio.NewReader(obj.ObjectReader)
	if _, err := io.Copy(fBuf, rBuf); err != nil {
		return errors.Wrap(err, "unable to read manta object")
	}

	if err := fBuf.Flush(); err != nil {
		return errors.Wrap(err, "unable to flush file")
	}

	if err := f.Close(); err != nil {
		return errors.Wrap(err, "unable to close file")
	}
	closed = true

	return nil
}

func editScrum(w io.Writer, c *scrumClient, scrumDate time.Time, user string) error {
	objectPath := path.Join("stor", "scrum", scrumDate.Format(scrumDateLayout), user)

	ctx, _ := context.WithTimeout(context.Background(), viper.GetDuration(configKeyMantaTimeout))
	start := time.Now()
	obj, err := c.Objects().Get(ctx, &storage.GetObjectInput{
		ObjectPath: objectPath,
	})
	elapsed := time.Now().Sub(start)
	log.Debug().Str("path", objectPath).Str("duration", elapsed.String()).Msg("GetObject")
	c.Histogram.RecordValue(float64(elapsed) / float64(time.Second))
	c.getCalls++
	if err != nil {
		return errors.Wrap(err, "unable to get manta object")
	}
	defer obj.ObjectReader.Close()

	body, err := ioutil.ReadAll(obj.ObjectReader)
	if err != nil {
		return errors.Wrap(err, "unable to read manta object")
	}

	w.Write(bytes.TrimSpace(body))
	w.Write([]byte("\n"))

	return nil
}
