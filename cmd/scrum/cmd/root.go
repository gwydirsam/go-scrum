package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "scrum",
	Short: "A command to post and read scrums",
	Long:  `scrum is used internally to post and read the daily scrum at Joyent.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Printf("MANTA_URL: %s\n", viper.Get("manta_url"))
	// 	mantaURL = viper.Get("manta_url").(string)
	// 	fmt.Printf("MANTA_KEY_ID: %s\n", viper.Get("manta_key_id"))
	// 	mantaURL = viper.Get("manta_key_id").(string)
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	{
		const key = configKeyMantaURL
		const longOpt, shortOpt = key, "U"
		const defaultValue = "https://us-east.manta.joyent.com"
		RootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "URL of the manta instance (default is $MANTA_URL)")
		viper.BindPFlag(key, RootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_URL")
	}

	{
		const key = configKeyMantaKeyID
		const longOpt, shortOpt = key, ""
		const defaultValue = ""
		RootCmd.PersistentFlags().StringP(longOpt, shortOpt, defaultValue, "SSH key fingerprint (default is $MANTA_KEY_ID)")
		viper.BindPFlag(key, RootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_KEY_ID")
	}

	{
		const key = configKeyMantaUser
		const longOpt, shortOpt = key, "u"
		RootCmd.PersistentFlags().StringP(longOpt, shortOpt, "$USER", "username to scrum as")
		viper.BindPFlag(key, RootCmd.PersistentFlags().Lookup(key))
		viper.BindEnv(key, "MANTA_USER")
	}

	{
		const key = configKeyTomorrow
		const longOpt, shortOpt = key, "t"
		RootCmd.PersistentFlags().BoolP(longOpt, shortOpt, false, "Scrum for tomorrow")
		viper.BindPFlag(key, RootCmd.PersistentFlags().Lookup(key))
	}
}

func CheckRequiredFlags(flags *pflag.FlagSet) error {
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
