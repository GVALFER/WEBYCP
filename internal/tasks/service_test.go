package tasks

import "testing"

func TestBuildRejectsUnsupportedScheduler(t *testing.T) {
	if _, err := build(ScheduledTask{Kind: Command}, "Hourly", "0 * * * *", "/usr/bin/true", "systemd", true); err == nil {
		t.Fatal("unsupported scheduler driver was accepted")
	}
}

func TestBuildTaskKindsAndCommands(t *testing.T) {
	for _, kind := range []Kind{"", "http", "backup", "maintenance"} {
		if _, err := build(ScheduledTask{Kind: kind}, "Task", "* * * * *", "/usr/bin/true", "crontab", true); err == nil {
			t.Errorf("accepted unsupported kind %q", kind)
		}
	}
	for _, command := range []string{"", "echo one\necho two", "date +%s", "echo\x00bad"} {
		if _, err := build(ScheduledTask{Kind: Command}, "Task", "* * * * *", command, "crontab", true); err == nil {
			t.Errorf("accepted unsafe command %q", command)
		}
	}
	value, err := build(ScheduledTask{Kind: Command}, " Hourly ", "0  * * * *", " /usr/bin/true ", "crontab", false)
	if err != nil || value.Name != "Hourly" || value.Schedule != "0 * * * *" || value.Command != "/usr/bin/true" || value.Enabled {
		t.Fatalf("task = %+v, err = %v", value, err)
	}
}
