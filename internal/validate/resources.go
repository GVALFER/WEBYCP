package validate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	robfigcron "github.com/robfig/cron/v3"
)

var databaseNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
var databaseSystemNamePattern = regexp.MustCompile(`^wcp_[a-f0-9]{8}_[a-z][a-z0-9_]{0,31}$`)
var cronParser = robfigcron.NewParser(
	robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow,
)

func DatabaseName(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if !databaseNamePattern.MatchString(normalized) {
		return "", &Error{
			Field: "name", Message: "Use 1–32 lowercase letters, digits, or underscores, starting with a letter",
		}
	}
	return normalized, nil
}

func DatabaseSystemName(value string) error {
	if !databaseSystemNamePattern.MatchString(value) || len(value) > 64 {
		return &Error{Field: "name", Message: "Invalid WEBYCP database name"}
	}
	return nil
}

func ResourceName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if utf8.RuneCountInString(normalized) < 1 || utf8.RuneCountInString(normalized) > 80 {
		return "", &Error{Field: "name", Message: "Use between 1 and 80 characters"}
	}
	for _, character := range normalized {
		if unicode.IsControl(character) {
			return "", &Error{Field: "name", Message: "Control characters are not allowed"}
		}
	}
	return normalized, nil
}

func CronCommand(value string) (string, error) {
	command := strings.TrimSpace(value)
	if command == "" || len(command) > 1000 || strings.ContainsAny(command, "\x00\r\n%") {
		return "", &Error{Field: "command", Message: "Use a single command of at most 1000 bytes"}
	}
	return command, nil
}

func CronSchedule(value string, allowEmpty bool) (string, error) {
	schedule := strings.Join(strings.Fields(value), " ")
	if schedule == "" && allowEmpty {
		return "", nil
	}
	if _, err := cronParser.Parse(schedule); err != nil {
		return "", &Error{Field: "schedule", Message: "Enter a valid five-field cron schedule"}
	}
	return schedule, nil
}
