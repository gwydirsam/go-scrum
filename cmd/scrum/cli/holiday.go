package cli

import (
	"fmt"
	"strings"
	"sync"
	"text/scanner"
	"time"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var joyentHolidays = map[string]string{
	"2018-01-01": `ca,uk,us: New Year's Day`,
	"2018-01-15": `us: Martin Luther King Day`,
	"2018-02-12": `ca: Family Day (BC)`,
	"2018-02-19": `ca,us: ca:"Family Day (AB, MB, ON, PE, SK)" us:"President's Day"`,
	"2018-03-30": `ca,uk: Good Friday`,
	"2018-04-02": `uk: Easter Monday`,
	"2018-04-13": `us: Wellbeing Day`,
	"2018-05-07": `uk: Early May bank holiday`,
	"2018-05-21": `ca: Victoria Day`,
	"2018-05-28": `uk: Spring bank holiday`,
	"2018-05-29": `us: Memorial Day`,
	"2018-06-25": `ca: National Holiday (QC)`,
	"2018-07-02": `ca: Canada Day, observed`,
	"2018-07-04": `us: Independence Day`,
	"2018-08-06": `ca: Civic Day (AB, BC, ON, NS, MB)`,
	"2018-08-27": `uk: Summer bank holiday`,
	"2018-09-03": `ca, us: Labor Day`,
	"2018-10-08": `ca: Thanksgiving Day`,
	"2018-11-12": `ca: Remembrance Day, observed (AB, BC, NS)`,
	"2018-11-22": `us: Thanksgiving Day`,
	"2018-11-23": `us: Day After Thanksgiving`,
	"2018-12-24": `us: Christmas Eve`,
	"2018-12-25": `ca,uk,us: Christmas Day`,
	"2018-12-26": `ca,uk: Boxing Day`,
}

// _Holiday is a statically typed string with implied meaning in the structure of the string itself.
//
// examples:
// "country: holiday name"
// "country1, country2: holiday name"
// `country1, country2: country1:"country1 holiday name" country2:"country 2 holiday name"`
type _Holiday string

// getCountries extracts the countries from a given _Holiday
func (h _Holiday) getCountries() []string {
	fieldsFunc := func(c rune) bool {
		return !unicode.IsLetter(c)
	}

	inObservance := strings.Split(string(h), ":")
	if len(inObservance) < 2 {
		panic(fmt.Sprintf(`invalid holiday (%+q): format must be: "country: holday name"`, string(h)))
	}

	countries := strings.FieldsFunc(inObservance[0], fieldsFunc)

	return countries
}

// getCountryHoliday returns the holiday name for the given country.
func (h _Holiday) getCountryHoliday(countryName string) (string, error) {
	countries := h.getCountries()
	numCountries := len(countries)

	holiday := strings.SplitN(string(h), ":", 2)
	if len(holiday) == 1 {
		return "", fmt.Errorf("unable to extract country from holiday: %q", h)
	}

	if numCountries == 1 {
		return holiday[1], nil
	}

	// Create a scanner to tokenize the input.
	var s scanner.Scanner
	s.Init(strings.NewReader(holiday[1]))
	s.Error = func(s *scanner.Scanner, msg string) {
		log.Warn().Str("holiday", string(h)).Int("character", s.Position.Column).Str("msg", msg).Msg("error scanning input")
	}
	s.Filename = "holiday"
	s.IsIdentRune = func(ch rune, i int) bool {
		return unicode.IsLetter(ch) && i >= 0
	}
	s.Mode = scanner.ScanIdents
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		// scanning for country:"name"
		country := s.TokenText()

		if s.Peek() != ':' {
			// malformed explicit format, fall back to returning the whole string
			break
		}

		tok = s.Scan() // Eat the :
		if tok == scanner.EOF {
			return holiday[1], fmt.Errorf("malformed holiday country separator: +q", h)
		}

		s.Mode = scanner.ScanStrings
		tok = s.Scan()
		if tok == scanner.EOF {
			return holiday[1], fmt.Errorf("malformed holiday string: +q", h)
		}

		if countryName == country {
			// We found our country's holiday
			return strings.Trim(s.TokenText(), `"`), nil
		}

		// Back to scanning characters
		s.Mode = scanner.ScanIdents
	}

	return strings.TrimSpace(holiday[1]), nil
}

var cachedHolidayMap map[time.Time]_Holiday
var holidayOnce sync.Once

// getHolidays reads a list of holidays and returns it as a map to the caller.
//
// NOTE(seanc@): The times are local to the caller of this utility.  This is a
// bit sketchy in terms of correctness, but good enough as long as the dates
// stored in the map match the dates from the perspective of the caller.
// Comparison across timezones?  Not so much.
//
// TODO(seanc@): build an inverted index and index the intervals or time spans
// and search the index for a matching holiday in a given country.
func getHolidays() map[time.Time]_Holiday {
	buildHolidayCache := func() {
		holidays := viper.GetStringMapString(configKeyHolidays)
		holidayMap := make(map[time.Time]_Holiday, len(holidays))

		// Always parse holidays local to their timezone.  Create a cache of countries
		// and their location.

		for dateStr, holiday := range holidays {
			date, err := getDateInLocation(dateStr)
			if err != nil {
				log.Warn().Err(err).Str("date", dateStr).Str("holiday", holiday).Msg("unable to parse holiday date")
				continue
			}
			holidayMap[date] = _Holiday(holiday)
		}

		cachedHolidayMap = holidayMap
	}

	holidayOnce.Do(buildHolidayCache)

	return cachedHolidayMap
}
