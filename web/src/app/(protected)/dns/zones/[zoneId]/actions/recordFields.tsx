"use client";

import { useFormContext, useWatch } from "react-hook-form";
import * as v from "valibot";
import { FormInput } from "@/components/form/formInput";
import { FormSelect } from "@/components/form/formSelect";
import type { DNSRecord, WriteDNSRecordRequest } from "@/contracts/types";

export const recordSchema = v.object({
    name: v.pipe(
        v.string(),
        v.trim(),
        v.nonEmpty("Enter a record name."),
        v.maxLength(253, "Use 253 characters or fewer."),
    ),
    type: v.picklist(["A", "AAAA", "CNAME", "MX", "TXT"]),
    content: v.pipe(
        v.string(),
        v.trim(),
        v.nonEmpty("Enter record content."),
        v.maxLength(1000, "Use 1000 characters or fewer."),
    ),
    ttl: v.pipe(
        v.number("Enter a TTL."),
        v.integer("Use a whole number."),
        v.minValue(60, "Use at least 60 seconds."),
        v.maxValue(86400, "Use at most 86400 seconds."),
    ),
    priority: v.pipe(
        v.number("Enter a priority."),
        v.integer("Use a whole number."),
        v.minValue(0, "Use a priority of 0 or more."),
        v.maxValue(65535, "Use a priority of 65535 or less."),
    ),
});

export type RecordValues = v.InferOutput<typeof recordSchema>;

export const recordValues = (record?: DNSRecord, defaultTtl = 3600): RecordValues => ({
    name: record?.name ?? "@",
    type: record?.type ?? "A",
    content: record?.content ?? "",
    ttl: record?.ttl ?? defaultTtl,
    priority: record?.priority ?? 0,
});

export const requestValues = (values: RecordValues) =>
    ({ ...values, priority: values.type === "MX" ? values.priority : 0 }) satisfies WriteDNSRecordRequest;

const RecordFields = () => {
    const { control } = useFormContext<RecordValues>();
    const type = useWatch({ control, name: "type" });

    return (
        <div className="grid gap-4 sm:grid-cols-2">
            <FormInput
                className="sm:col-span-2"
                name="name"
                label="Record name"
                maxLength={253}
                required
            />
            <FormSelect
                name="type"
                label="Type"
                options={["A", "AAAA", "CNAME", "MX", "TXT"].map((value) => ({
                    id: value,
                    name: value,
                }))}
                required
            />
            <FormInput name="ttl" label="TTL" type="number" min={60} max={86400} required />
            <FormInput
                className="sm:col-span-2"
                inputClassName="font-mono"
                name="content"
                label="Content"
                maxLength={1000}
                required
            />
            {type === "MX" && (
                <FormInput
                    name="priority"
                    label="Priority"
                    type="number"
                    min={0}
                    max={65535}
                    required
                />
            )}
        </div>
    );
};

export default RecordFields;
