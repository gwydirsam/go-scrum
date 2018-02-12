package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/gops/agent"
	"github.com/gwydirsam/go-scrum/cmd/scrum/internal/buildtime"
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
	Example: `  $ scrum get             # Get my scrum for today
  $ scrum get -a          # Get everyone's scrum
  $ scrum set -i today.md # Set my scrum using today.md
  $ scrum list            # List scrummers for the day`,
	Args: cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Perform input validation

		logLevel, err := initLogLevels()
		if err != nil {
			return errors.Wrap(err, "unable to initialize log levels")
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
			if logLevel != _LogLevelDebug {
				stdLogger.SetOutput(ioutil.Discard)
			} else {
				stdLogger.SetOutput(zlog)
			}
		}

		// Always enable the agent
		if err := agent.Listen(nil); err != nil {
			log.Fatal().Err(err).Msg("unable to start gops agent")
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}

	return nil
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	_, _ = initLogLevels()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug().Err(err).Msg("unable to read config file")
		} else {
			log.Warn().Err(err).Msg("unable to read config file")
		}
	}
}

func init() {
	// Initialize viper so that when ew call initLogLevels() we can pull a value
	// from a config file.
	viper.SetConfigName(buildtime.PROGNAME)
	viper.AddConfigPath(path.Join("$HOME", ".config", buildtime.PROGNAME))
	viper.AddConfigPath(".")

	// os.Stderr isn't guaranteed to be thread-safe, wrap in a sync writer.  Files
	// are guaranteed to be safe, terminals are not.
	w := zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
	}
	zlog := zerolog.New(zerolog.SyncWriter(w)).With().Timestamp().Logger()

	zerolog.DurationFieldUnit = time.Microsecond
	zerolog.DurationFieldInteger = true
	zerolog.TimeFieldFormat = logTimeFormat
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Logger = zlog

	stdlog.SetFlags(0)
	stdlog.SetOutput(zlog)

	{
		const (
			key          = configKeyCountry
			longOpt      = "country"
			shortOpt     = "C"
			defaultValue = "us"
			description  = "Country holiday schedule"
		)

		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const key = configKeyHolidays
		defaultValue := joyentHolidays

		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key          = configKeyLogLevel
			longOpt      = "log-level"
			shortOpt     = "l"
			defaultValue = "INFO"
			description  = "Change the log level being sent to stdout"
		)

		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)

		// Initialize the log levels immediately.  initLogLevels() will be called
		// again later during the standard initialization procedure.
		_, _ = initLogLevels()
	}

	{
		const (
			key         = configKeyLogFormat
			longOpt     = "log-format"
			shortOpt    = "F"
			description = `Specify the log format ("auto", "zerolog", or "human")`
		)

		defaultValue := _LogFormatAuto.String()
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key               = configKeyLogStats
			longOpt, shortOpt = "stats", "S"
			defaultValue      = true
			description       = "Log Manta client latency stats on exit"
		)
		rootCmd.PersistentFlags().BoolP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const (
			key         = configKeyLogTermColor
			longOpt     = "use-color"
			shortOpt    = ""
			description = "Use ASCII colors"
		)

		defaultValue := false
		if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
			defaultValue = true
		}

		rootCmd.PersistentFlags().BoolP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const key = configKeyMantaAccount
		const longOpt, shortOpt = "manta-account", "A"
		const defaultValue = "$MANTA_USER"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "Manta account name")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.BindEnv(key, "MANTA_ACCOUNT")
	}

	{
		const key = configKeyMantaKeyID
		const longOpt, shortOpt = "manta-key-id", ""
		const defaultValue = ""
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "SSH key fingerprint (default is $MANTA_KEY_ID)")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.BindEnv(key, "MANTA_KEY_ID")
	}

	{
		const (
			key          = configKeyMantaTimeout
			longOpt      = "manta-timeout"
			shortOpt     = "T"
			description  = "Manta API timeout"
			defaultValue = 3 * time.Second
		)

		rootCmd.PersistentFlags().DurationP(longOpt, shortOpt, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.SetDefault(key, defaultValue)
	}

	{
		const key = configKeyMantaURL
		const longOpt, shortOpt = "manta-url", "E"
		const defaultValue = "https://us-east.manta.joyent.com"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "URL of the Manta instance (default is $MANTA_URL)")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.BindEnv(key, "MANTA_URL")
	}

	{
		const key = configKeyMantaUser
		const longOpt, shortOpt = "manta-user", "U"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, "$MANTA_USER", "Manta username to scrum as")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.BindEnv(key, "MANTA_USER")
	}

	{
		const key = configKeyScrumAccount
		const longOpt, shortOpt = "scrum-account", "B"
		const defaultValue = "Joyent_Dev"
		rootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "Manta account for scrum board/files")
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longOpt))
		viper.BindEnv(key, "SCRUM_ACCOUNT")
	}

	{
		const (
			key          = configKeyUseUTC
			longName     = "utc"
			shortName    = "Z"
			defaultValue = false
			description  = "Display times in UTC"
		)

		rootCmd.PersistentFlags().BoolP(longName, shortName, defaultValue, description)
		viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(longName))
		viper.SetDefault(key, defaultValue)
	}

	cobra.OnInitialize(initConfig)
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

type _LogLevel int

const (
	_LogLevelBegin _LogLevel = iota - 2
	_LogLevelDebug
	_LogLevelInfo // Default, zero-initialized value
	_LogLevelWarn
	_LogLevelError
	_LogLevelFatal

	_LogLevelEnd
)

func (f _LogLevel) String() string {
	switch f {
	case _LogLevelDebug:
		return "debug"
	case _LogLevelInfo:
		return "info"
	case _LogLevelWarn:
		return "warn"
	case _LogLevelError:
		return "error"
	case _LogLevelFatal:
		return "fatal"
	default:
		panic(fmt.Sprintf("unknown log level: %d", f))
	}
}

func logLevels() []_LogLevel {
	levels := make([]_LogLevel, 0, _LogLevelEnd-_LogLevelBegin)
	for i := _LogLevelBegin + 1; i < _LogLevelEnd; i++ {
		levels = append(levels, i)
	}

	return levels
}

func logLevelsStr() []string {
	intLevels := logLevels()
	levels := make([]string, 0, len(intLevels))
	for _, lvl := range intLevels {
		levels = append(levels, lvl.String())
	}
	return levels
}

func initLogLevels() (logLevel _LogLevel, err error) {
	switch strLevel := strings.ToLower(viper.GetString(configKeyLogLevel)); strLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logLevel = _LogLevelDebug
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		logLevel = _LogLevelInfo
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		logLevel = _LogLevelWarn
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		logLevel = _LogLevelError
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
		logLevel = _LogLevelFatal
	default:
		return _LogLevelDebug, fmt.Errorf("unsupported error level: %q (supported levels: %s)", logLevel,
			strings.Join(logLevelsStr(), " "))
	}

	return logLevel, nil
}
