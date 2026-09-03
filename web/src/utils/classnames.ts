type ClassValue = false | null | string | undefined;

export const cn = (...values: ClassValue[]) => values.filter(Boolean).join(" ");
