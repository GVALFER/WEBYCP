"use client";

import { ErrorMessage, Input, Label, TextField } from "@heroui/react";
import type { ComponentProps } from "react";
import { useController, useFormContext } from "react-hook-form";

type Props = Omit<
    ComponentProps<typeof Input>,
    "className" | "defaultValue" | "name" | "onBlur" | "onChange" | "value"
> & {
    className?: string;
    inputClassName?: string;
    label: string;
    name: string;
    required?: boolean;
};

export const FormInput = ({
    className,
    inputClassName,
    label,
    name,
    required = false,
    ...props
}: Props) => {
    const { control } = useFormContext();
    const {
        field: { name: fieldName, onBlur, onChange, ref, value },
        fieldState,
    } = useController({ control, name });

    return (
        <TextField
            className={className}
            name={fieldName}
            isInvalid={fieldState.invalid}
            isRequired={required}
            fullWidth
        >
            <Label>{label}</Label>
            <Input
                {...props}
                ref={ref}
                className={inputClassName}
                value={value ?? ""}
                onBlur={onBlur}
                onChange={onChange}
            />
            <ErrorMessage>{fieldState.error?.message}</ErrorMessage>
        </TextField>
    );
};
