"use client";

import { ErrorMessage, Label, ListBox, Select } from "@heroui/react";
import { useController, useFormContext } from "react-hook-form";

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
        <Select
            ref={ref}
            className={className}
            name={fieldName}
            selectedKey={value || null}
            placeholder={empty}
            isDisabled={!options.length}
            isInvalid={fieldState.invalid}
            isRequired={required}
            fullWidth
            onBlur={onBlur}
            onSelectionChange={(key) => {
                const selected = String(key);
                onChange(selected);
                onValueChange?.(selected);
            }}
        >
            <Label>{label}</Label>
            <Select.Trigger>
                <Select.Value />
                <Select.Indicator />
            </Select.Trigger>
            <Select.Popover>
                <ListBox>
                    {options.map((option) => (
                        <ListBox.Item key={option.id} id={option.id} textValue={option.name}>
                            {option.name}
                            <ListBox.ItemIndicator />
                        </ListBox.Item>
                    ))}
                </ListBox>
            </Select.Popover>
            <ErrorMessage>{fieldState.error?.message}</ErrorMessage>
        </Select>
    );
};
