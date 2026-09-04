"use client";

import { cn } from "../utils/classnames";

type Option = {
    id: string;
    name: string;
};

type Props = {
    label: string;
    value: string;
    options: Option[];
    onChange: (value: string) => void;
    className?: string;
    empty?: string;
    required?: boolean;
};

export const SelectField = ({
    label,
    value,
    options,
    onChange,
    className,
    empty = "No options available",
    required = false,
}: Props) => (
    <label className={cn("block space-y-2 text-sm", className)}>
        <span className="font-medium">{label}</span>
        <select
            className="h-10 w-full rounded-xl border border-border bg-field-background px-3 outline-none transition focus:border-accent"
            required={required}
            value={value}
            onChange={(event) => onChange(event.currentTarget.value)}
        >
            {!options.length && <option value="">{empty}</option>}
            {options.map((option) => (
                <option key={option.id} value={option.id}>
                    {option.name}
                </option>
            ))}
        </select>
    </label>
);
