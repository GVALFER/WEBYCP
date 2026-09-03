package validate

import (
	"fmt"
	"net"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

var (
	idPattern         = regexp.MustCompile(`^[a-f0-9]{32}$`)
	systemUserPattern = regexp.MustCompile(`^wcp_[a-f0-9]{12}$`)
)

const MaxDomainAliases = 100

type Error struct {
	Field   string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func Email(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized || len(normalized) > 254 {
		return "", &Error{Field: "email", Message: "Enter a valid email address"}
	}

	return normalized, nil
}

func Name(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	length := utf8.RuneCountInString(normalized)
	if length < 2 || length > 80 {
		return "", &Error{Field: "name", Message: "Use between 2 and 80 characters"}
	}

	return normalized, nil
}

func Password(value string) error {
	length := utf8.RuneCountInString(value)
	if length < 12 || length > 128 {
		return &Error{Field: "password", Message: "Use between 12 and 128 characters"}
	}

	return nil
}

func AccountName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	length := utf8.RuneCountInString(normalized)
	if length < 2 || length > 80 {
		return "", &Error{Field: "name", Message: "Use between 2 and 80 characters"}
	}
	for _, character := range normalized {
		if unicode.IsControl(character) {
			return "", &Error{Field: "name", Message: "Control characters are not allowed"}
		}
	}

	return normalized, nil
}

func SystemUser(value string) error {
	if !systemUserPattern.MatchString(value) {
		return &Error{Field: "systemUser", Message: "Invalid WEBYCP system user"}
	}

	return nil
}

func ID(field, value string) error {
	if !idPattern.MatchString(value) {
		return &Error{Field: field, Message: "Invalid resource ID"}
	}

	return nil
}

func Domain(value string) (string, error) {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	ascii, err := idna.Lookup.ToASCII(normalized)
	if err != nil || len(ascii) > 253 || net.ParseIP(ascii) != nil {
		return "", &Error{Field: "name", Message: "Enter a valid domain name"}
	}
	labels := strings.Split(ascii, ".")
	if len(labels) < 2 {
		return "", &Error{Field: "name", Message: "Enter a valid domain name"}
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", &Error{Field: "name", Message: "Enter a valid domain name"}
		}
		for _, character := range label {
			if character != '-' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return "", &Error{Field: "name", Message: "Enter a valid domain name"}
			}
		}
	}

	return ascii, nil
}

func DomainAliases(primary string, aliases []string) ([]string, error) {
	name, err := Domain(primary)
	if err != nil || name != primary {
		return nil, &Error{Field: "name", Message: "Domain name is not normalized"}
	}
	if len(aliases) > MaxDomainAliases {
		return nil, &Error{Field: "aliases", Message: "Too many domain aliases"}
	}
	seen := map[string]struct{}{primary: {}}
	result := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		normalized, err := Domain(alias)
		if err != nil || normalized != alias {
			return nil, &Error{Field: "aliases", Message: "Domain alias is not normalized"}
		}
		if _, exists := seen[alias]; exists {
			return nil, &Error{Field: "aliases", Message: "Domain aliases must be unique"}
		}
		seen[alias] = struct{}{}
		result = append(result, alias)
	}
	sort.Strings(result)
	return result, nil
}
