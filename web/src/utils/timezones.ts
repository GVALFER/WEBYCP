export const TIMEZONES = [
    { label: "UTC (Coordinated Universal Time)", value: "UTC" },
    { label: "Europe/Lisbon (Portugal)", value: "Europe/Lisbon" },
    { label: "Europe/London (United Kingdom)", value: "Europe/London" },
    { label: "Europe/Madrid (Spain)", value: "Europe/Madrid" },
    { label: "Europe/Paris (France)", value: "Europe/Paris" },
    { label: "Europe/Berlin (Germany)", value: "Europe/Berlin" },
    { label: "Europe/Amsterdam (Netherlands)", value: "Europe/Amsterdam" },
    { label: "Europe/Rome (Italy)", value: "Europe/Rome" },
    { label: "Europe/Athens (Greece)", value: "Europe/Athens" },
    { label: "Europe/Istanbul (Turkey)", value: "Europe/Istanbul" },
    { label: "Europe/Moscow (Russia)", value: "Europe/Moscow" },
    { label: "Africa/Casablanca (Morocco)", value: "Africa/Casablanca" },
    { label: "Africa/Johannesburg (South Africa)", value: "Africa/Johannesburg" },
    { label: "America/Sao_Paulo (Brazil)", value: "America/Sao_Paulo" },
    {
        label: "America/Argentina/Buenos_Aires (Argentina)",
        value: "America/Argentina/Buenos_Aires",
    },
    { label: "America/New_York (USA - Eastern)", value: "America/New_York" },
    { label: "America/Chicago (USA - Central)", value: "America/Chicago" },
    { label: "America/Denver (USA - Mountain)", value: "America/Denver" },
    { label: "America/Los_Angeles (USA - Pacific)", value: "America/Los_Angeles" },
    { label: "America/Mexico_City (Mexico)", value: "America/Mexico_City" },
    { label: "America/Toronto (Canada)", value: "America/Toronto" },
    { label: "Asia/Dubai (UAE)", value: "Asia/Dubai" },
    { label: "Asia/Kolkata (India)", value: "Asia/Kolkata" },
    { label: "Asia/Bangkok (Thailand)", value: "Asia/Bangkok" },
    { label: "Asia/Singapore (Singapore)", value: "Asia/Singapore" },
    { label: "Asia/Hong_Kong (Hong Kong)", value: "Asia/Hong_Kong" },
    { label: "Asia/Tokyo (Japan)", value: "Asia/Tokyo" },
    { label: "Asia/Seoul (South Korea)", value: "Asia/Seoul" },
    { label: "Asia/Shanghai (China)", value: "Asia/Shanghai" },
    { label: "Australia/Sydney (Australia)", value: "Australia/Sydney" },
    { label: "Pacific/Auckland (New Zealand)", value: "Pacific/Auckland" },
];

export const getTimezone = (tz: string) => {
    return (
        TIMEZONES.find((timezone) => timezone.value === tz) || {
            label: "UTC (Coordinated Universal Time)",
            value: "UTC",
        }
    );
};

export const isValidTimezone = (tz: string | undefined) => {
    if (typeof tz !== "string") return false;
    return TIMEZONES.some((item) => item.value === tz);
};
