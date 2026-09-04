"use client";

import { useController, useFormContext } from "react-hook-form";
import { cn } from "@/utils/classnames";

type Props = {
    className?: string;
    label: string;
    name: string;
};

export const FormCheckbox = ({ className, label, name }: Props) => {
    const { control } = useFormContext();
    const {
        field: { name: fieldName, onBlur, onChange, ref, value },
        fieldState,
    } = useController({ control, name });

    return (
        <label className={cn("flex items-center gap-3 text-sm", className)}>
            <input
                ref={ref}
                type="checkbox"
                name={fieldName}
                checked={value === true}
                onBlur={onBlur}
                onChange={(event) => onChange(event.currentTarget.checked)}
            />
            {label}
            {fieldState.error?.message && (
                <span className="text-xs text-danger" role="alert">
                    {fieldState.error.message}
                </span>
            )}
        </label>
    );
};
