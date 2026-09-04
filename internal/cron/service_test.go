package cronjob

import "testing"

func TestBuildRejectsUnsupportedScheduler(t *testing.T) {
	if _, err := build(CronJob{}, "Hourly", "0 * * * *", "/usr/bin/true", "systemd", true); err == nil {
		t.Fatal("unsupported scheduler driver was accepted")
	}
}
