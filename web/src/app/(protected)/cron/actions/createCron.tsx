"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, CronJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
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
    const { mutate } = useSWRConfig();
    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];
    const [pending, startTransition] = useTransition();
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
                    await Promise.all([mutate("cron-jobs"), mutate("jobs")]);
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
            <form className="panel-card h-fit p-6" onSubmit={form.handleSubmit(handleSubmit)}>
                <h2 className="text-base font-semibold">Add cron job</h2>
                <FormSelect
                    className="mt-5"
                    name="accountId"
                    label="Hosting account"
                    options={options}
                    required
                />
                <FormInput className="mt-4" name="name" label="Name" maxLength={80} required />
                <FormInput
                    className="mt-4"
                    inputClassName="font-mono"
                    name="schedule"
                    label="Schedule"
                    maxLength={100}
                    required
                />
                <FormInput
                    className="mt-4"
                    inputClassName="font-mono"
                    name="command"
                    label="Command"
                    maxLength={1_000}
                    required
                />
                <Button
                    className="mt-5"
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={!accountId || pending}
                >
                    <Plus className="size-4" />
                    Add cron job
                </Button>
            </form>
        </Form>
    );
};

export default CreateCron;
