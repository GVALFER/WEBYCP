"use client";

import { useController, useFormContext } from "react-hook-form";
import { cn } from "@/utils/classnames";

export type FormOption = {
    id: string;
    name: string;
};

type Props = {
    className?: string;
    empty?: string;
    label: string;
    name: string;
    onValueChange?: (value: string) => void;
    options: FormOption[];
    required?: boolean;
};

export const FormSelect = ({
    className,
    empty = "No options available",
    label,
    name,
    onValueChange,
    options,
    required = false,
}: Props) => {
    const { control } = useFormContext();
    const {
        field: { name: fieldName, onBlur, onChange, ref, value },
        fieldState,
    } = useController({ control, name });

    return (
        <label className={cn("block space-y-2 text-sm", className)}>
            <span className="font-medium">{label}</span>
            <select
                ref={ref}
                className={cn(
                    "h-10 w-full rounded-xl border bg-field-background px-3 outline-none transition focus:border-accent",
                    fieldState.invalid ? "border-danger" : "border-border",
                )}
                name={fieldName}
                required={required}
                value={value ?? ""}
                onBlur={onBlur}
                onChange={(event) => {
                    onChange(event.currentTarget.value);
                    onValueChange?.(event.currentTarget.value);
                }}
            >
                {!options.length && <option value="">{empty}</option>}
                {options.map((option) => (
                    <option key={option.id} value={option.id}>
                        {option.name}
                    </option>
                ))}
            </select>
            {fieldState.error?.message && (
                <span className="block text-xs text-danger" role="alert">
                    {fieldState.error.message}
                </span>
            )}
        </label>
    );
};
