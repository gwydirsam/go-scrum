package cmd

const (
	dateInputFormat = "2006-01-02"

	configKeyGetOptAll = "get.opt-all"

	configKeyInitFilename = "init.config-file"

	configKeyInputDate = "date"

	configKeyLogFormat    = "log.format"
	configKeyLogLevel     = "log.level"
	configKeyLogTermColor = "log.use-color"

	configKeyMantaAccount = "manta.account"
	configKeyMantaKeyID   = "manta.key-id"
	configKeyMantaURL     = "manta.url"
	configKeyMantaUser    = "manta.user"

	configKeySetFilename     = "set.input-filename"
	configKeySetForce        = "set.force"
	configKeySetNumDays      = "set.num-days"
	configKeySetSickDays     = "set.sick-days"
	configKeySetVacationDays = "set.vacation-days"

	configKeyTomorrow = "tomorrow"
	configKeyUsername = "username"

	scrumDateLayout = "2006/01/02"
)

var ignoreMap = map[string]struct{}{
	"all": struct{}{},
}
