package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	MinTTL = 60
	MaxTTL = 86400
)

func ValidateSettings(value Settings) (Settings, error) {
	if value.DefaultTTL < MinTTL || value.DefaultTTL > MaxTTL {
		return Settings{}, &validate.Error{Field: "defaultTtl", Message: "Use a TTL between 60 and 86400 seconds"}
	}
	primary := strings.TrimSpace(value.PrimaryNameserver)
	secondary := strings.TrimSpace(value.SecondaryNameserver)
	if primary == "" && secondary == "" {
		value.PrimaryNameserver, value.SecondaryNameserver = "", ""
		return value, nil
	}
	if primary == "" || secondary == "" {
		return Settings{}, &validate.Error{Field: "primaryNameserver", Message: "Configure both nameservers"}
	}
	var err error
	primary, err = validate.Domain(primary)
	if err != nil {
		return Settings{}, &validate.Error{Field: "primaryNameserver", Message: "Enter a valid nameserver"}
	}
	secondary, err = validate.Domain(secondary)
	if err != nil {
		return Settings{}, &validate.Error{Field: "secondaryNameserver", Message: "Enter a valid nameserver"}
	}
	if primary == secondary {
		return Settings{}, &validate.Error{Field: "secondaryNameserver", Message: "Nameservers must be different"}
	}
	value.PrimaryNameserver, value.SecondaryNameserver = primary, secondary
	return value, nil
}

func NormalizeRecord(value Record, zone string) (Record, error) {
	name, err := recordName(value.Name, zone)
	if err != nil {
		return Record{}, err
	}
	kind := strings.ToUpper(strings.TrimSpace(value.Type))
	if kind != "A" && kind != "AAAA" && kind != "CNAME" && kind != "MX" && kind != "TXT" {
		return Record{}, &validate.Error{Field: "type", Message: "Choose a supported DNS record type"}
	}
	if value.TTL < MinTTL || value.TTL > MaxTTL {
		return Record{}, &validate.Error{Field: "ttl", Message: "Use a TTL between 60 and 86400 seconds"}
	}
	content, priority, err := recordContent(kind, value.Content, value.Priority)
	if err != nil {
		return Record{}, err
	}
	if kind == "CNAME" && name == zone {
		return Record{}, &validate.Error{Field: "name", Message: "The zone apex cannot be a CNAME"}
	}
	value.Name, value.Type, value.Content = name, kind, content
	value.Priority = priority
	return value, nil
}

func ProviderContent(value Record) string {
	switch value.Type {
	case "CNAME":
		return value.Content + "."
	case "MX":
		return fmt.Sprintf("%d %s.", value.Priority, value.Content)
	case "TXT":
		return strconv.Quote(value.Content)
	default:
		return value.Content
	}
}

func recordName(value, zone string) (string, error) {
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	if name == "@" {
		return zone, nil
	}
	if !strings.Contains(name, ".") {
		name += "." + zone
	}
	if name != zone && !strings.HasSuffix(name, "."+zone) {
		return "", &validate.Error{Field: "name", Message: "Record name must belong to the selected zone"}
	}
	labels := strings.Split(name, ".")
	for index, label := range labels {
		if label == "*" && index == 0 {
			continue
		}
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", &validate.Error{Field: "name", Message: "Enter a valid DNS record name"}
		}
		for _, character := range label {
			letter := character >= 'a' && character <= 'z'
			digit := character >= '0' && character <= '9'
			if character != '-' && character != '_' && !letter && !digit {
				return "", &validate.Error{Field: "name", Message: "Enter a valid DNS record name"}
			}
		}
	}
	return name, nil
}

func recordContent(kind, value string, priority int64) (string, int64, error) {
	content := strings.TrimSpace(value)
	switch kind {
	case "A":
		address := net.ParseIP(content)
		if address == nil || address.To4() == nil {
			return "", 0, &validate.Error{Field: "content", Message: "Enter a valid IPv4 address"}
		}
		return address.To4().String(), 0, nil
	case "AAAA":
		address := net.ParseIP(content)
		if address == nil || address.To4() != nil {
			return "", 0, &validate.Error{Field: "content", Message: "Enter a valid IPv6 address"}
		}
		return address.String(), 0, nil
	case "CNAME", "MX":
		target, err := validate.Domain(content)
		if err != nil {
			return "", 0, &validate.Error{Field: "content", Message: "Enter a valid target hostname"}
		}
		if kind == "MX" {
			if priority < 0 || priority > 65535 {
				return "", 0, &validate.Error{Field: "priority", Message: "Use a priority between 0 and 65535"}
			}
			return target, priority, nil
		}
		return target, 0, nil
	case "TXT":
		if content == "" || len(content) > 1000 || strings.ContainsAny(content, "\x00\r\n") {
			return "", 0, &validate.Error{Field: "content", Message: "Use a single line of at most 1000 bytes"}
		}
		return content, 0, nil
	}
	return "", 0, &validate.Error{Field: "type", Message: "Choose a supported DNS record type"}
}
