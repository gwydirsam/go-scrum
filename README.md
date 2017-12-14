# `scrum`

A Joyent scrum utility.

## Installation

1. Install `scrum`: `go get -v -u github.com/gwydirsam/go-scrum/cmd/scrum`
2. Make sure `scrum` is in your path:

    ```
    $ export GOPATH=`go env GOPATH`
    $ export PATH=$GOPATH/bin:$PATH
    $ which scrum
    ~/go/bin/scrum
    ```

3. Initialize a config file or set environment variables.
  1. To use the `scrum` config file, use run `scrum init` to generate a default
     config file (default path: `$HOME/.config/scrum/scrum.toml`).
  2. Set environment variables.  See the #direnv section below.

Regardless of which config format used, `MANTA_KEY_ID` is generated by one of the
following:

```
$ ssh-keygen -E md5 -lf ~/.ssh/id_rsa.pub | awk '{print $2}' | cut -d : -f 2-
$ ssh-keygen -E md5 -lf ~/.ssh/id_ecdsa.pub | awk '{print $2}' | cut -d : -f 2-
```

## Usage

```
$ scrum -h
scrum is used internally to post and read the daily scrum at Joyent.

Usage:
  scrum [command]

Available Commands:
  get         Get scrum information
  help        Help about any command
  init        Generate an initial scrum configuration file
  list        List scrum information
  set         Set scrum information

Flags:
  -h, --help                   help for scrum
  -F, --log-format string      Specify the log format ("auto", "zerolog", or "human") (default "auto")
  -l, --log-level string       Change the log level being sent to stdout (default "INFO")
  -A, --manta-account string   Manta account name (default "Joyent_Dev")
      --manta-key-id string    SSH key fingerprint (default is $MANTA_KEY_ID)
  -E, --manta-url string       URL of the Manta instance (default is $MANTA_URL) (default "https://us-east.manta.joyent.com")
  -U, --manta-user string      Manta username to scrum as (default "$MANTA_USER")
      --use-color              Use ASCII colors

Use "scrum [command] --help" for more information about a command.
```

### `scrum get` Usage

```
$ scrum get
Vacation until 2017/11/24
$ scrum get -h
Get scrum information, either for yourself (or teammates)

Usage:
  scrum get [flags]

Examples:
  $ scrum get                      # Get my scrum for today
  $ scrum get -t -u other.username # Get other.username's scrum for tomorrow

Flags:
  -a, --all           Get scrum for all users
  -D, --date string   Date for scrum (default "2017-12-11")
  -h, --help          help for get
  -t, --tomorrow      Get scrum for the next day
  -P, --use-pager     Use a pager to read the output (defaults to $PAGER, less(1), or more(1)) (default true)
  -u, --user string   Get scrum for specified user (default "$USER")
  -Z, --utc           Get mtime data in UTC
  -y, --yesterday     Get scrum for yesterday

Global Flags:
  -F, --log-format string      Specify the log format ("auto", "zerolog", or "human") (default "auto")
  -l, --log-level string       Change the log level being sent to stdout (default "INFO")
  -A, --manta-account string   Manta account name (default "Joyent_Dev")
      --manta-key-id string    SSH key fingerprint (default is $MANTA_KEY_ID)
  -E, --manta-url string       URL of the Manta instance (default is $MANTA_URL) (default "https://us-east.manta.joyent.com")
  -U, --manta-user string      Manta username to scrum as (default "$MANTA_USER")
      --use-color              Use ASCII colors
```

### `scrum set` Usage

```
$ scrum set -h
Set scrum information, either for yourself (or teammates)

Usage:
  scrum set [flags]

Examples:
  $ scrum set -i today.md                         # Set my scrum using today.md
  $ scrum set -u other.username -t -i tomorrow.md # Set other.username's scrum for tomorrow

Flags:
  -D, --date string     Date for scrum (default "2017-12-11")
  -d, --days uint       Recycle scrum update for N days
  -i, --file string     File to read scrum from
  -f, --force           Force overwrite of any present scrum
  -h, --help            help for set
  -s, --sick uint       Sick leave for N days
  -t, --tomorrow        Set scrum for the next day
  -u, --user string     Set scrum for specified user (default "$USER")
  -v, --vacation uint   Vacation for N days

Global Flags:
  -F, --log-format string      Specify the log format ("auto", "zerolog", or "human") (default "auto")
  -l, --log-level string       Change the log level being sent to stdout (default "INFO")
  -A, --manta-account string   Manta account name (default "Joyent_Dev")
      --manta-key-id string    SSH key fingerprint (default is $MANTA_KEY_ID)
  -E, --manta-url string       URL of the Manta instance (default is $MANTA_URL) (default "https://us-east.manta.joyent.com")
  -U, --manta-user string      Manta username to scrum as (default "$MANTA_USER")
      --use-color              Use ASCII colors
```

### `scrum list` Usage

```
$ scrum list -h
List scrum information for the day

Usage:
  scrum list [flags]

Examples:
  $ scrum list                      # List scrummers for the day
  $ scrum list -t

Flags:
  -a, --all           List all metadata details (default true)
  -D, --date string   Date for scrum (default "2017-12-11")
  -h, --help          help for list
  -t, --tomorrow      List scrums for the next day
  -1, --usernames     List usernames only
  -Z, --utc           List mtime data in UTC
  -y, --yesterday     List scrum for yesterday

Global Flags:
  -F, --log-format string      Specify the log format ("auto", "zerolog", or "human") (default "auto")
  -l, --log-level string       Change the log level being sent to stdout (default "INFO")
  -A, --manta-account string   Manta account name (default "Joyent_Dev")
      --manta-key-id string    SSH key fingerprint (default is $MANTA_KEY_ID)
  -E, --manta-url string       URL of the Manta instance (default is $MANTA_URL) (default "https://us-east.manta.joyent.com")
  -U, --manta-user string      Manta username to scrum as (default "$MANTA_USER")
      --use-color              Use ASCII colors
```

## `direnv`

1. Install [`direnv`](https://github.com/direnv/direnv) and integrate into your
   shell.
2. Populate a `.envrc` file:

    ```
    export MANTA_USER=my-manta-username
    export MANTA_URL=https://us-east.manta.joyent.com
    export MANTA_KEY_ID=00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff
    ```

3. `direnv allow` the directory containing the `.envrc` file (e.g. `cd ~/src/joyent/engdoc && direnv allow`).

