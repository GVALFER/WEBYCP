import { useSession } from "@/providers/SessionProvider";
import { isValidTimezone } from "@/utils/timezones";
import dayjs from "dayjs";
import timezone from "dayjs/plugin/timezone";
import utc from "dayjs/plugin/utc";

export type TimezoneFormat =
    | "DD/MM/YYYY HH:mm"
    | "DD/MM/YYYY"
    | "DD MMM YYYY"
    | "DD MMMM YYYY"
    | "DD MMM YYYY HH:mm"
    | "DD MMM"
    | "YYYY"
    | "DD/MM"
    | "DD/HH:mm"
    | "DD/MM HH:mm"
    | "HH:mm"
    | "DD"
    | "ddd"
    | "MMMM YYYY"
    | "dddd, MMMM YYYY"
    | "h:mmA";

export type TimezoneFormatter = (date: dayjs.ConfigType, format?: TimezoneFormat) => string;

dayjs.extend(utc);
dayjs.extend(timezone);

const DEFAULT_TZ = "UTC";

export const useTimezone = () => {
    const session = useSession();

    const tz: string = isValidTimezone(session?.timezone || undefined)
        ? (session?.timezone as string)
        : DEFAULT_TZ;

    const offset = dayjs().tz(tz).format("Z");

    const dt: TimezoneFormatter = (
        date: dayjs.ConfigType,
        format: TimezoneFormat = "DD/MM/YYYY HH:mm",
    ) => {
        if (!date) return "";
        return dayjs(date).tz(tz).format(format);
    };

    const daysSince = (date: dayjs.ConfigType) => {
        const due = dayjs(date).tz(tz);
        const now = dayjs().tz(tz);
        return now.diff(due, "day");
    };

    const daysUntil = (date: dayjs.ConfigType) => {
        const due = dayjs(date).tz(tz);
        const now = dayjs().tz(tz);
        return due.diff(now, "day");
    };

    const iso = (date: dayjs.ConfigType) => {
        return dayjs(date).tz(tz).toISOString();
    };

    return { dt, daysSince, tz, offset, daysUntil, iso };
};
