"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { Pencil } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import type { DNSSettings, UpdateDNSSettingsRequest } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

const optionalDomain = v.union([
    v.literal(""),
    v.pipe(
        v.string(),
        v.trim(),
        v.maxLength(253, "Use 253 characters or fewer."),
        v.regex(
            /^(?:[a-z\d](?:[a-z\d-]{0,61}[a-z\d])?\.)+[a-z\d](?:[a-z\d-]{0,61}[a-z\d])?$/i,
            "Enter a valid hostname.",
        ),
    ),
]);

const schema = v.object({
    primaryNameserver: optionalDomain,
    secondaryNameserver: optionalDomain,
    defaultTtl: v.pipe(
        v.number("Enter a default TTL."),
        v.integer("Use a whole number."),
        v.minValue(60, "Use at least 60 seconds."),
        v.maxValue(86400, "Use at most 86400 seconds."),
    ),
});

type Values = v.InferOutput<typeof schema>;

const EditNameservers = ({ settings }: { settings: DNSSettings }) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();
    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: settings,
    });

    const handleOpenChange = useCallback(
        (value: boolean) => {
            setOpen(value);
            if (value) form.reset(settings);
        },
        [form, settings],
    );

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    const response = await api
                        .put("dns/settings", {
                            json: values satisfies UpdateDNSSettingsRequest,
                        })
                        .json<DNSSettings>();
                    await mutate("dns/settings", response, false);
                    setOpen(false);
                    toast.success("Nameserver defaults updated");
                } catch (error) {
                    toast.danger("Nameserver defaults could not be updated", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [mutate],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Edit nameserver defaults"
                description="Configure both nameservers, or leave both empty to disable new zone creation."
                triggerLabel="Edit nameserver defaults"
                triggerIcon={<Pencil className="size-4" aria-hidden="true" />}
                triggerText="Edit"
                triggerVariant="secondary"
                submitLabel="Save defaults"
                pending={pending}
                onOpenChange={handleOpenChange}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormInput name="primaryNameserver" label="Primary nameserver" maxLength={253} />
                <FormInput
                    name="secondaryNameserver"
                    label="Secondary nameserver"
                    maxLength={253}
                />
                <FormInput
                    name="defaultTtl"
                    label="Default TTL"
                    type="number"
                    min={60}
                    max={86400}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default EditNameservers;
