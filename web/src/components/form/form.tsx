"use client";

import type { ReactNode } from "react";
import {
    FormProvider,
    type FieldValues,
    type UseFormReturn,
} from "react-hook-form";

type Props<
    TValues extends FieldValues,
    TContext = unknown,
    TOutput extends FieldValues | undefined = TValues,
> = UseFormReturn<TValues, TContext, TOutput> & {
    children: ReactNode;
};

export const Form = <
    TValues extends FieldValues,
    TContext = unknown,
    TOutput extends FieldValues | undefined = TValues,
>({ children, ...form }: Props<TValues, TContext, TOutput>) => (
    <FormProvider {...form}>{children}</FormProvider>
);
