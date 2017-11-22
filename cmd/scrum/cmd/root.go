package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"strings"
	"time"

	"github.com/gwydirsam/go-scrum/cmd/scrum/buildtime"
	isatty "github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var stdLogger *stdlog.Logger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "scrum",
	SilenceErrors: true,
	Short:         "A command to post and read scrums",
	Long:          `scrum is used internally to post and read the daily scrum at Joyent.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Printf("MANTA_URL: %s\n", viper.Get("manta_url"))
	// 	mantaURL = viper.Get("manta_url").(string)
	// 	fmt.Printf("MANTA_KEY_ID: %s\n", viper.Get("manta_key_id"))
	// 	mantaURL = viper.Get("manta_key_id").(string)
	// },

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Perform input validation

		var logLevel string
		switch logLevel = strings.ToUpper(viper.GetString(configKeyLogLevel)); logLevel {
		case "DEBUG":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "INFO":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "WARN":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "ERROR":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		case "FATAL":
			zerolog.SetGlobalLevel(zerolog.FatalLevel)
		default:
			// FIXME(seanc@): move the supported log levels into a global constant
			return fmt.Errorf("unsupported error level: %q (supported levels: %s)", logLevel,
				strings.Join([]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}, " "))
		}

		// zerolog was initialized with sane defaults.  Re-initialize logging with
		// user-supplied configuration parameters
		{
			// os.Stderr isn't guaranteed to be thread-safe, wrap in a sync writer.
			// Files are guaranteed to be safe, terminals are not.
			var logWriter io.Writer
			if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
				logWriter = zerolog.SyncWriter(os.Stderr)
			} else {
				logWriter = os.Stderr
			}

			logFmt, err := getLogFormat()
			if err != nil {
				return errors.Wrap(err, "unable to parse log format")
			}

			if logFmt == _LogFormatAuto {
				if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
					logFmt = _LogFormatHuman
				} else {
					logFmt = _LogFormatZerolog
				}
			}

			var zlog zerolog.Logger
			switch logFmt {
			case _LogFormatZerolog:
				zlog = zerolog.New(logWriter).With().Timestamp().Logger()
			case _LogFormatHuman:
				useColor := viper.GetBool(configKeyLogTermColor)
				w := zerolog.ConsoleWriter{
					Out:     logWriter,
					NoColor: !useColor,
				}
				zlog = zerolog.New(w).With().Timestamp().Logger()
			default:
				return fmt.Errorf("unsupported log format: %q")
			}

			log.Logger = zlog

			stdlog.SetFlags(0)
			stdlog.SetOutput(zlog)
			stdLogger = &stdlog.Logger{}
			if logLevel != "DEBUG" {
				stdLogger.SetOutput(ioutil.Discard)
			} else {
				stdLogger.SetOutput(zlog)
			}
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName(buildtime.PROGNAME)
}

func init() {
	cobra.OnInitialize(initConfig)

	// os.Stderr isn't guaranteed to be thread-safe, wrap in a sync writer.  Files
	// are guaranteed to be safe, terminals are not.
	w := zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
	}
	zlog := zerolog.New(zerolog.SyncWriter(w)).With().Timestamp().Logger()

	cobra.OnInitialize(initConfig)

	zerolog.DurationFieldUnit = time.Microsecond
	zerolog.DurationFieldInteger = true
	zerolog.TimeFieldFormat = logTimeFormat
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Logger = zlog

	stdlog.SetFlags(0)
	stdlog.SetOutput(zlog)

	{
		const (
			key          = configKeyLogLevel
			longName     = "log-level"
			shortName    = "l"
			defaultValue = "INFO"
			description  = "Change the log level being sent to stdout"
		)

		rootCmd.PersistentFlags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key         = configKeyLogFormat
			longName    = "log-format"
			shortName   = "F"
			description = `Specify the log format ("auto", "zerolog", or "human")`
		)

		defaultValue := _LogFormatAuto.String()
		rootCmd.PersistentFlags().StringP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key         = configKeyLogTermColor
			longName    = "use-color"
			shortName   = ""
			description = "Use ASCII colors"
		)

		defaultValue := false
		if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
			defaultValue = true
		}

		rootCmd.PersistentFlags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	{
		const key = configKeyMantaURL
		const longOpt, shortOpt = key, "U"
		const defaultValue = "https://us-east.manta.joyent.com"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "URL of the manta instance (default is $MANTA_URL)")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_URL")
	}

	{
		const key = configKeyMantaKeyID
		const longOpt, shortOpt = key, ""
		const defaultValue = ""
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "SSH key fingerprint (default is $MANTA_KEY_ID)")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_KEY_ID")
	}

	{
		const key = configKeyMantaUser
		const longOpt, shortOpt = key, "u"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, "$USER", "username to scrum as")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_USER")
	}

	{
		const key = configKeyTomorrow
		const longOpt, shortOpt = key, "t"
		rootCmd.PersistentFlags().BoolP(longOpt, shortOpt, false, "Scrum for tomorrow")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(key))
	}
}

func checkRequiredFlags(flags *pflag.FlagSet) error {
	requiredError := false
	flagName := ""

	flags.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"

		if flagRequired && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return errors.New("Required flag `" + flagName + "` has not been set")
	}

	return nil
}

const (
	// Use a log format that resembles time.RFC3339Nano but includes all trailing
	// zeros so that we get fixed-width logging.
	logTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"
)

type _LogFormat uint

const (
	_LogFormatAuto _LogFormat = iota
	_LogFormatZerolog
	_LogFormatHuman
)

func (f _LogFormat) String() string {
	switch f {
	case _LogFormatAuto:
		return "auto"
	case _LogFormatZerolog:
		return "zerolog"
	case _LogFormatHuman:
		return "human"
	default:
		panic(fmt.Sprintf("unknown log format: %d", f))
	}
}

func getLogFormat() (_LogFormat, error) {
	switch logFormat := strings.ToLower(viper.GetString(configKeyLogFormat)); logFormat {
	case "auto":
		return _LogFormatAuto, nil
	case "json", "zerolog":
		return _LogFormatZerolog, nil
	case "human":
		return _LogFormatHuman, nil
	default:
		return _LogFormatAuto, fmt.Errorf("unsupported log format: %q", logFormat)
	}
}
