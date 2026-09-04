"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, CronJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { commandField, nameField, scheduleField } from "@/utils/validation";

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    name: nameField,
    schedule: v.pipe(scheduleField, v.nonEmpty("Enter a schedule.")),
    command: commandField,
});

type FormValues = v.InferOutput<typeof formSchema>;

type CreateCronProps = {
    accounts: AccountListResponse;
};

const CreateCron = ({ accounts }: CreateCronProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<AccountListResponse>(pageKey("accounts", { page: 1, size: 100 }), {
        fallbackData: accounts,
    });

    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            accountId: options[0]?.id ?? "",
            name: "",
            schedule: "0 * * * *",
            command: "",
        },
    });

    const accountId = useWatch({ control: form.control, name: "accountId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post("cron-jobs", { json: { ...values, enabled: true } })
                        .json<CronJobResponse>();
                    form.reset({
                        accountId: values.accountId,
                        name: "",
                        schedule: values.schedule,
                        command: "",
                    });
                    await mutate((key) => isPageKey(key, "cron-jobs", "accounts", "jobs"));
                    setOpen(false);
                    toast.success("Cron job queued for creation");
                } catch (error) {
                    toast.danger("Action failed", {
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
                title="Add cron job"
                description="Runs a command as the hosting account from its home directory."
                triggerLabel="Add cron job"
                submitLabel="Add cron job"
                pending={pending}
                submitDisabled={!accountId}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="accountId"
                    label="Hosting account"
                    options={options}
                    empty="No active accounts"
                    required
                />
                <FormInput name="name" label="Name" maxLength={80} required />
                <FormInput
                    inputClassName="font-mono"
                    name="schedule"
                    label="Schedule"
                    maxLength={100}
                    required
                />
                <FormInput
                    inputClassName="font-mono"
                    name="command"
                    label="Command"
                    maxLength={1_000}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateCron;
