package services

import (
	"context"
	"testing"
	"time"
)

type repository struct {
	value Settings
}

func (r *repository) ServiceSettings(context.Context) (Settings, error) {
	return r.value, nil
}

func (r *repository) UpdateServiceSettings(_ context.Context, value Settings) (Settings, error) {
	r.value = value
	return value, nil
}

func TestServiceSettings(t *testing.T) {
	repository := &repository{value: Settings{Defaults: testDefaults(), UpdatedAt: time.Now().UTC()}}
	service := NewService(repository)

	value, err := service.Update(context.Background(), testDefaults())
	if err != nil {
		t.Fatal(err)
	}
	if value.Defaults != testDefaults() || value.UpdatedAt.IsZero() {
		t.Fatalf("settings = %+v", value)
	}
}

func TestServiceSettingsRejectUnsupportedDefaults(t *testing.T) {
	value := testDefaults()
	value.WebDriver = "apache"
	if _, err := NewService(&repository{}).Update(context.Background(), value); err == nil {
		t.Fatal("unsupported default was accepted")
	}
}

func testDefaults() Defaults {
	return Defaults{
		WebDriver: Nginx, RuntimeDriver: PHPFPM, RuntimeVersion: PHP83,
		DatabaseDriver: MySQL, SchedulerDriver: Crontab, BackupDriver: Local,
	}
}
