package capability

import (
	"context"
	"io/fs"
	"os"
	"time"

	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/services"
)

const checkTimeout = 5 * time.Second

type Observer interface {
	Observe(context.Context) services.Capabilities
}

type Checker interface {
	Health(context.Context) error
}

type System struct {
	run  func(context.Context, string, ...string) error
	stat func(string) (fs.FileInfo, error)
	dns  Checker
}

func New(dns Checker) *System {
	return &System{run: execx.Run, stat: os.Stat, dns: dns}
}

func (s *System) Observe(ctx context.Context) services.Capabilities {
	return services.Capabilities{
		Webservers: []services.Capability{
			s.command(ctx, services.Nginx, "", "nginx", "-t"),
		},
		Runtimes: []services.Capability{
			s.command(ctx, services.PHPFPM, services.PHP83, "php-fpm8.3", "-t"),
		},
		Databases: []services.Capability{
			s.command(ctx, services.MySQL, "", "mysqladmin", "ping", "--silent"),
		},
		Schedulers: []services.Capability{
			s.command(ctx, services.Crontab, "", "systemctl", "is-active", "--quiet", "cron"),
		},
		Backups: []services.Capability{
			s.directory(services.Local, "/var/backups/webycp"),
		},
		DNS: []services.Capability{
			s.check(ctx, services.PowerDNS, s.dns),
		},
	}
}

func (s *System) check(ctx context.Context, driver string, checker Checker) services.Capability {
	status := services.Unavailable
	if checker != nil {
		checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
		defer cancel()
		if checker.Health(checkCtx) == nil {
			status = services.Healthy
		}
	}
	return services.Capability{Driver: driver, Status: status}
}

func (s *System) command(
	ctx context.Context, driver, version, name string, args ...string,
) services.Capability {
	checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	status := services.Healthy
	if err := s.run(checkCtx, name, args...); err != nil {
		status = services.Unavailable
	}
	return services.Capability{Driver: driver, Version: version, Status: status}
}

func (s *System) directory(driver, path string) services.Capability {
	status := services.Healthy
	info, err := s.stat(path)
	if err != nil || !info.IsDir() {
		status = services.Unavailable
	}
	return services.Capability{Driver: driver, Status: status}
}
