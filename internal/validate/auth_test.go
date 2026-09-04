package validate

import "testing"

func TestUsername(t *testing.T) {
	username, err := Username(" Owner.Name ")
	if err != nil {
		t.Fatal(err)
	}
	if username != "owner.name" {
		t.Fatalf("username = %q", username)
	}

	for _, value := range []string{"ab", "owner name", "-owner"} {
		if _, err := Username(value); err == nil {
			t.Errorf("Username(%q) accepted invalid value", value)
		}
	}
}

func TestTimezone(t *testing.T) {
	timezone, err := Timezone(" Europe/Lisbon ")
	if err != nil {
		t.Fatal(err)
	}
	if timezone != "Europe/Lisbon" {
		t.Fatalf("timezone = %q", timezone)
	}

	for _, value := range []string{"", "Europe/Invalid", "not a timezone"} {
		if _, err := Timezone(value); err == nil {
			t.Errorf("Timezone(%q) accepted invalid value", value)
		}
	}
}
