"use client";

import { Checkbox, ErrorMessage, Label } from "@heroui/react";
import { useController, useFormContext } from "react-hook-form";

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
        <Checkbox
            ref={ref}
            className={className}
            name={fieldName}
            isInvalid={fieldState.invalid}
            isSelected={value === true}
            onBlur={onBlur}
            onChange={onChange}
        >
            <Checkbox.Content>
                <Checkbox.Control>
                    <Checkbox.Indicator />
                </Checkbox.Control>
                <Label>{label}</Label>
            </Checkbox.Content>
            <ErrorMessage>{fieldState.error?.message}</ErrorMessage>
        </Checkbox>
    );
};
