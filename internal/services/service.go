package services

import (
	"context"
	"time"

	"github.com/GVALFER/WEBYCP/internal/validate"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	return s.repository.ServiceSettings(ctx)
}

func (s *Service) Update(ctx context.Context, defaults Defaults) (Settings, error) {
	if err := ValidateDefaults(defaults); err != nil {
		return Settings{}, err
	}
	return s.repository.UpdateServiceSettings(ctx, Settings{
		Defaults: defaults, UpdatedAt: time.Now().UTC(),
	})
}

func ValidateDefaults(value Defaults) error {
	checks := []struct {
		field string
		value string
		want  string
	}{
		{"webDriver", value.WebDriver, Nginx},
		{"runtimeDriver", value.RuntimeDriver, PHPFPM},
		{"runtimeVersion", value.RuntimeVersion, PHP83},
		{"databaseDriver", value.DatabaseDriver, MySQL},
		{"schedulerDriver", value.SchedulerDriver, Crontab},
		{"backupDriver", value.BackupDriver, Local},
	}
	for _, check := range checks {
		if check.value != check.want {
			return &validate.Error{
				Field: check.field, Message: "The selected service is not supported",
			}
		}
	}
	return nil
}
