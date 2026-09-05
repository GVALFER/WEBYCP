package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Zone struct {
	ID          string
	Name        string
	Nameservers []string
}

type RecordSet struct {
	Name    string
	Type    string
	TTL     int64
	Records []string
}

type Driver interface {
	Health(context.Context) error
	EnsureZone(context.Context, Zone) error
	DeleteZone(context.Context, Zone) error
	SyncRecordSets(context.Context, Zone, []RecordSet) error
}

func ValidateZone(value Zone) error {
	if err := ValidateZoneIdentity(value); err != nil {
		return err
	}
	if len(value.Nameservers) != 2 {
		return fmt.Errorf("invalid DNS zone")
	}
	first, firstErr := validate.Domain(value.Nameservers[0])
	second, secondErr := validate.Domain(value.Nameservers[1])
	if firstErr != nil || secondErr != nil || first != value.Nameservers[0] ||
		second != value.Nameservers[1] || first == second {
		return fmt.Errorf("invalid DNS nameservers")
	}
	return nil
}

func ValidateZoneIdentity(value Zone) error {
	if err := validate.ID("id", value.ID); err != nil {
		return err
	}
	name, err := validate.Domain(value.Name)
	if err != nil || name != value.Name {
		return fmt.Errorf("invalid DNS zone")
	}
	return nil
}

func ValidateRecordSets(zone Zone, values []RecordSet) error {
	if err := ValidateZoneIdentity(zone); err != nil {
		return err
	}
	if len(values) < 1 || len(values) > 2 {
		return fmt.Errorf("invalid DNS record set count")
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !recordName(value.Name, zone.Name) || !recordType(value.Type) {
			return fmt.Errorf("invalid DNS record set")
		}
		key := value.Name + "\x00" + value.Type
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate DNS record set")
		}
		seen[key] = struct{}{}
		if len(value.Records) == 0 {
			if value.TTL != 0 {
				return fmt.Errorf("deleted DNS record set has a TTL")
			}
			continue
		}
		if value.TTL < 60 || value.TTL > 86400 || len(value.Records) > 100 {
			return fmt.Errorf("invalid DNS record set TTL or size")
		}
		for _, record := range value.Records {
			if record == "" || len(record) > 2048 || strings.ContainsAny(record, "\x00\r\n") {
				return fmt.Errorf("invalid DNS record content")
			}
		}
	}
	return nil
}

func recordType(value string) bool {
	return value == "A" || value == "AAAA" || value == "CNAME" || value == "MX" || value == "TXT"
}

func recordName(value, zone string) bool {
	if value != strings.ToLower(strings.TrimSuffix(value, ".")) ||
		(value != zone && !strings.HasSuffix(value, "."+zone)) {
		return false
	}
	for index, label := range strings.Split(value, ".") {
		if label == "*" && index == 0 {
			continue
		}
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			letter := character >= 'a' && character <= 'z'
			digit := character >= '0' && character <= '9'
			if character != '-' && character != '_' && !letter && !digit {
				return false
			}
		}
	}
	return true
}
