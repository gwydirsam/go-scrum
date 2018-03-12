package cmd

import (
	"os"
	"time"

	"github.com/circonus-labs/circonusllhist"
	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
	"github.com/joyent/triton-go/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// scrumClient wraps a StorateClient and a Histogram
type scrumClient struct {
	*storage.StorageClient

	// Time per operation (us)
	*circonusllhist.Histogram

	// Count of each operation type
	deleteCalls uint64
	getCalls    uint64
	listCalls   uint64
	putCalls    uint64
}

func (sc *scrumClient) dumpMantaClientStats() {
	if !viper.GetBool(configKeyLogStats) {
		return
	}

	log.Info().
		Str("tp99", (time.Duration(sc.Histogram.ValueAtQuantile(0.99)*float64(time.Second))).String()).
		Str("tp95", (time.Duration(sc.Histogram.ValueAtQuantile(0.95)*float64(time.Second))).String()).
		Str("tp90", (time.Duration(sc.Histogram.ValueAtQuantile(0.90)*float64(time.Second))).String()).
		Str("max", (time.Duration(sc.Histogram.Max()*float64(time.Second))).String()).
		Str("mean", (time.Duration(sc.Histogram.Mean()*float64(time.Second))).String()).
		Str("min", (time.Duration(sc.Histogram.Min()*float64(time.Second))).String()).
		Str("total", (time.Duration(sc.Histogram.ApproxSum()*float64(time.Second))).String()).
		Uint64("get-calls", sc.getCalls).
		Uint64("list-calls", sc.listCalls).
		Uint64("put-calls", sc.putCalls).
		Msg("stats")
}

// getDateInLocation takes a given date string and parses it according to whether or not
// the user requested UTC or Local timezone processing.
func getDateInLocation(dateStr string) (date time.Time, err error) {
	if viper.GetBool(configKeyUseUTC) {
		date, err = time.Parse(dateInputFormat, dateStr)
	} else {
		localLocation, err := time.LoadLocation("Local")
		if err != nil {
			return time.Now(), errors.Wrap(err, "unable to load local timezone information")
		}

		date, err = time.ParseInLocation(dateInputFormat, dateStr, localLocation)
	}
	if err != nil {
		return time.Now(), errors.Wrap(err, "unable to parse date")
	}

	return date, nil
}

// getNextWeekday returns the next weekday.
func getNextWeekday(scrumDate time.Time) time.Time {
	return getWeekday(scrumDate, true)
}

// getPreviousWeekday returns the previous weekday.
func getPreviousWeekday(scrumDate time.Time) time.Time {
	return getWeekday(scrumDate, false)
}

func getScrumClient() (*scrumClient, error) {
	input := authentication.SSHAgentSignerInput{
		KeyID:       viper.GetString(configKeyMantaKeyID),
		AccountName: interpolateMantaUserEnvVar(viper.GetString(configKeyMantaAccount)),
	}
	sshKeySigner, err := authentication.NewSSHAgentSigner(input)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create new SSH agent signer")
	}

	tsc, err := storage.NewClient(&triton.ClientConfig{
		MantaURL:    viper.GetString(configKeyMantaURL),
		AccountName: interpolateMantaUserEnvVar(viper.GetString(configKeyScrumAccount)),
		Signers:     []authentication.Signer{sshKeySigner},
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a new manta client")
	}

	return &scrumClient{
		StorageClient: tsc,
		Histogram:     circonusllhist.New(),
	}, nil
}

// getWeekday is the internal helper function that either adds or subtracts a
// weekday and tests to see if the next day in the sequence is a holiday or not
// for the given country.
//
// TODO: teach getWeekday to take in to consideration a vacation schedule.
func getWeekday(scrumDate time.Time, nextDay bool) time.Time {
	holidays := getHolidays()

	myCountry := viper.GetString(configKeyCountry)

NEXT_DATE:
	for {
		if nextDay {
			scrumDate = scrumDate.AddDate(0, 0, 1)
		} else {
			scrumDate = scrumDate.AddDate(0, 0, -1)
		}

		switch scrumDate.Weekday() {
		case time.Monday, time.Tuesday, time.Wednesday,
			time.Thursday, time.Friday:

			holiday, found := holidays[scrumDate]
			if !found {
				return scrumDate
			}

			countries := holiday.getCountries()
			for _, country := range countries {
				// Search for a date until my country is not in observance of a holiday.
				if country == myCountry {
					holidayName, err := holiday.getCountryHoliday(country)
					if err != nil {
						log.Warn().Err(err).Msg("unable to get a country's specific holiday")
					}

					log.Info().Str("country", country).Str("holiday", holidayName).Str("date", scrumDate.Format(dateInputFormat)).Msg("skipping holiday")
					continue NEXT_DATE
				}
			}

			// myCountry isn't observing a holiday on this day
			return scrumDate
		}
	}

	panic("unpossible")
}

func interpolateMantaUserEnvVar(val string) string {
	switch val {
	case "$MANTA_USER":
		return os.Getenv("MANTA_USER")
	}

	return val
}

func interpolateUserEnvVar(val string) string {
	switch val {
	case "$USER":
		return os.Getenv("USER")
	}

	return val
}
