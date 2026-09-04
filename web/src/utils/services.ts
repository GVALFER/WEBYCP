export const SERVICE_OPTIONS = {
    webDriver: [{ id: "nginx", name: "Nginx" }],
    runtimeDriver: [{ id: "phpfpm", name: "PHP-FPM" }],
    runtimeVersion: [{ id: "8.3", name: "PHP 8.3" }],
    databaseDriver: [{ id: "mysql", name: "MySQL" }],
    schedulerDriver: [{ id: "crontab", name: "Crontab" }],
    backupDriver: [{ id: "local", name: "Local storage" }],
} as const;
