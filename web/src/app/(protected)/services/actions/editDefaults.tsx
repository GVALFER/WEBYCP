"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { ServiceDefaults, ServiceSettings } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { SERVICE_OPTIONS } from "@/utils/services";

const schema = v.object({
    webDriver: v.literal("nginx"),
    runtimeDriver: v.literal("phpfpm"),
    runtimeVersion: v.literal("8.3"),
    databaseDriver: v.literal("mysql"),
    schedulerDriver: v.literal("crontab"),
    backupDriver: v.literal("local"),
});

type Values = v.InferOutput<typeof schema>;

const EditDefaults = ({ settings }: { settings: ServiceSettings }) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();

    const { mutate } = useSWRConfig();
    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: settings.defaults,
    });

    const handleOpenChange = useCallback(
        (value: boolean) => {
            setOpen(value);
            if (value) form.reset(settings.defaults);
        },
        [form, settings.defaults],
    );

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    const response = await api
                        .put("service-settings", { json: values satisfies ServiceDefaults })
                        .json<ServiceSettings>();
                    await mutate("service-settings", response, false);
                    setOpen(false);
                    toast.success("Service defaults updated");
                } catch (error) {
                    toast.danger("Defaults could not be updated", {
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
                title="Edit service defaults"
                description="These values preselect drivers when a new resource is created."
                triggerLabel="Edit defaults"
                submitLabel="Save defaults"
                pending={pending}
                size="lg"
                onOpenChange={handleOpenChange}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <div className="grid gap-4 sm:grid-cols-2">
                    <FormSelect
                        name="webDriver"
                        label="Web server"
                        options={SERVICE_OPTIONS.webDriver}
                        required
                    />
                    <FormSelect
                        name="runtimeDriver"
                        label="Runtime"
                        options={SERVICE_OPTIONS.runtimeDriver}
                        required
                    />
                    <FormSelect
                        name="runtimeVersion"
                        label="Runtime version"
                        options={SERVICE_OPTIONS.runtimeVersion}
                        required
                    />
                    <FormSelect
                        name="databaseDriver"
                        label="Database"
                        options={SERVICE_OPTIONS.databaseDriver}
                        required
                    />
                    <FormSelect
                        name="schedulerDriver"
                        label="Scheduler"
                        options={SERVICE_OPTIONS.schedulerDriver}
                        required
                    />
                    <FormSelect
                        name="backupDriver"
                        label="Backup storage"
                        options={SERVICE_OPTIONS.backupDriver}
                        required
                    />
                </div>
            </FormModal>
        </Form>
    );
};

export default EditDefaults;
