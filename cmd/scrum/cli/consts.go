package cli

const (
	dateInputFormat = "2006-01-02"

	configKeyGetAll       = "get.all"
	configKeyGetHighlight = "highlight"
	configKeyGetInputDate = "get.date"
	configKeyGetTomorrow  = "get.tomorrow"
	configKeyGetYesterday = "get.yesterday"

	configKeyHolidays = "holidays"
	configKeyCountry  = "general.country"
	configKeyUsePager = "general.use-pager"
	configKeyUseUTC   = "general.utc"

	configKeyScrumAccount  = "scrum.manta-account"
	configKeyScrumUsername = "scrum.username"

	configKeyInitFilename = "init.config-file"

	configKeyListInputDate = "list.date"
	configKeyListTomorrow  = "list.tomorrow"
	configKeyListUsers     = "list.mode"
	configKeyListUsersAll  = "list.opt-all"
	configKeyListUsersOne  = "list.opt-one"
	configKeyListYesterday = "list.yesterday"

	configKeyLogFormat    = "log.format"
	configKeyLogLevel     = "log.level"
	configKeyLogStats     = "log.stats"
	configKeyLogTermColor = "log.use-color"

	configKeyMantaAccount = "manta.account"
	configKeyMantaKeyID   = "manta.key-id"
	configKeyMantaTimeout = "manta.timeout"
	configKeyMantaURL     = "manta.url"
	configKeyMantaUser    = "manta.user"

	configKeySetFilename     = "set.input-filename"
	configKeySetForce        = "set.force"
	configKeySetInputDate    = "set.date"
	configKeySetNumDays      = "set.num-days"
	configKeySetSickDays     = "set.sick-days"
	configKeySetTomorrow     = "set.tomorrow"
	configKeySetUnlinkDay    = "set.unlink-day"
	configKeySetVacationDays = "set.vacation-days"
	configKeySetYesterday    = "set.yesterday"

	mtimeFormat   = "2006-01-02 15:04:05"
	mtimeFormatTZ = "2006-01-02 15:04:05 MST"

	scrumDateLayout = "2006/01/02"
)

type _UsernameAction int

const (
	_Show _UsernameAction = iota
	_Ignore
)

var usernameActionMap = map[string]_UsernameAction{
	"all":          _Ignore,
	"all1999.html": _Ignore,
	"rollup":       _Ignore,
}
