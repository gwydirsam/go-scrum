package cmd

const (
	dateInputFormat = "2006-01-02"

	configKeyGetAll       = "get.all"
	configKeyGetInputDate = "get.date"
	configKeyGetTomorrow  = "get.tomorrow"
	configKeyGetUTC       = "get.utc"
	configKeyGetUsePager  = "get.use-pager"
	configKeyGetUsername  = "get.username"
	configKeyGetYesterday = "get.yesterday"

	configKeyInitFilename = "init.config-file"

	configKeyListInputDate = "list.date"
	configKeyListTomorrow  = "list.tomorrow"
	configKeyListUsers     = "list.mode"
	configKeyListUsersAll  = "list.opt-all"
	configKeyListUsersOne  = "list.opt-one"
	configKeyListUsersUTC  = "list.opt-utc"
	configKeyListYesterday = "list.yesterday"

	configKeyLogFormat    = "log.format"
	configKeyLogLevel     = "log.level"
	configKeyLogTermColor = "log.use-color"

	configKeyMantaAccount = "manta.account"
	configKeyMantaKeyID   = "manta.key-id"
	configKeyMantaURL     = "manta.url"
	configKeyMantaUser    = "manta.user"

	configKeySetFilename     = "set.input-filename"
	configKeySetForce        = "set.force"
	configKeySetInputDate    = "set.date"
	configKeySetNumDays      = "set.num-days"
	configKeySetSickDays     = "set.sick-days"
	configKeySetTomorrow     = "set.tomorrow"
	configKeySetUnlinkDay    = "set.unlink-day"
	configKeySetUsername     = "set.username"
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
