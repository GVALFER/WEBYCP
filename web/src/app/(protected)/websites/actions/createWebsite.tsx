"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, ServiceDefaults, WebsiteJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { SERVICE_OPTIONS } from "@/utils/services";
import { domainField, nameField } from "@/utils/validation";

const schema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose an account.")),
    name: nameField,
    primaryDomain: domainField,
    webDriver: v.literal("nginx"),
    runtimeDriver: v.literal("phpfpm"),
    runtimeVersion: v.literal("8.3"),
});
type Values = v.InferOutput<typeof schema>;

type Props = {
    accounts: AccountListResponse;
    defaults: ServiceDefaults;
};

const CreateWebsite = ({ accounts, defaults }: Props) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const options = accounts.items.filter((item) => item.status === "active" && item.enabled);

    const form = useForm<Values>({
        resolver: valibotResolver(schema),
        defaultValues: {
            accountId: options[0]?.id ?? "",
            name: "",
            primaryDomain: "",
            webDriver: defaults.webDriver,
            runtimeDriver: defaults.runtimeDriver,
            runtimeVersion: defaults.runtimeVersion,
        },
    });

    const handleSubmit = useCallback(
        (values: Values) => {
            startTransition(async () => {
                try {
                    await api
                        .post("websites", {
                            json: {
                                ...values,
                                kind: "php",
                            },
                        })
                        .json<WebsiteJobResponse>();
                    form.reset({
                        accountId: values.accountId,
                        name: "",
                        primaryDomain: "",
                        webDriver: values.webDriver,
                        runtimeDriver: values.runtimeDriver,
                        runtimeVersion: values.runtimeVersion,
                    });
                    await mutate((key) =>
                        isPageKey(key, "websites", "website-domains", "accounts", "jobs"),
                    );
                    setOpen(false);
                    toast.success("Website queued for creation");
                } catch (error) {
                    toast.danger("Website action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [form, mutate],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Create website"
                description="Creates a PHP website with Nginx and an isolated document root."
                triggerLabel="Create website"
                submitLabel="Create website"
                pending={pending}
                submitDisabled={!options.length}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="accountId"
                    label="Account"
                    options={options}
                    empty="No active accounts"
                    required
                />
                <FormInput
                    name="name"
                    label="Website name"
                    placeholder="Customer website"
                    maxLength={80}
                    required
                />
                <FormInput
                    name="primaryDomain"
                    label="Primary domain"
                    placeholder="example.com"
                    maxLength={253}
                    required
                />
                <div className="grid gap-4 sm:grid-cols-3">
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
                        label="Version"
                        options={SERVICE_OPTIONS.runtimeVersion}
                        required
                    />
                </div>
            </FormModal>
        </Form>
    );
};

export default CreateWebsite;
