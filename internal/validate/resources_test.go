package validate

import "testing"

func TestDatabaseNames(t *testing.T) {
	name, err := DatabaseName("App_Data")
	if err != nil || name != "app_data" {
		t.Fatalf("name = %q, error = %v", name, err)
	}
	for _, value := range []string{"1app", "app-name", "app`; DROP DATABASE mysql"} {
		if _, err := DatabaseName(value); err == nil {
			t.Fatalf("DatabaseName(%q) should fail", value)
		}
	}
}

func TestCronValidation(t *testing.T) {
	if value, err := CronSchedule(" 0   3 * * * ", false); err != nil || value != "0 3 * * *" {
		t.Fatalf("schedule = %q, error = %v", value, err)
	}
	for _, value := range []string{"", "60 * * * *", "* * * *"} {
		if _, err := CronSchedule(value, false); err == nil {
			t.Fatalf("CronSchedule(%q) should fail", value)
		}
	}
	if _, err := CronCommand("date +%s"); err == nil {
		t.Fatal("cron percent expansion should be rejected")
	}
}
