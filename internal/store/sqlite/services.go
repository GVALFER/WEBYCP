package sqlite

import (
	"context"

	"github.com/GVALFER/WEBYCP/internal/services"
	"github.com/GVALFER/WEBYCP/internal/store/sqlite/dbgen"
)

func (s *Store) ServiceSettings(ctx context.Context) (services.Settings, error) {
	row, err := s.queries.GetServiceSettings(ctx)
	if err != nil {
		return services.Settings{}, err
	}
	return serviceSettingsValue(row), nil
}

func (s *Store) UpdateServiceSettings(
	ctx context.Context, value services.Settings,
) (services.Settings, error) {
	row, err := s.queries.UpdateServiceSettings(ctx, dbgen.UpdateServiceSettingsParams{
		WebDriver: value.Defaults.WebDriver, RuntimeDriver: value.Defaults.RuntimeDriver,
		RuntimeVersion: value.Defaults.RuntimeVersion, DatabaseDriver: value.Defaults.DatabaseDriver,
		SchedulerDriver: value.Defaults.SchedulerDriver, BackupDriver: value.Defaults.BackupDriver,
		UpdatedAt: timeValue(value.UpdatedAt),
	})
	if err != nil {
		return services.Settings{}, err
	}
	return serviceSettingsValue(row), nil
}

func serviceSettingsValue(row dbgen.ServiceSetting) services.Settings {
	return services.Settings{
		Defaults: services.Defaults{
			WebDriver: row.WebDriver, RuntimeDriver: row.RuntimeDriver,
			RuntimeVersion: row.RuntimeVersion, DatabaseDriver: row.DatabaseDriver,
			SchedulerDriver: row.SchedulerDriver, BackupDriver: row.BackupDriver,
		},
		UpdatedAt: timeFrom(row.UpdatedAt),
	}
}
