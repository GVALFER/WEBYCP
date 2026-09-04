"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormCheckbox } from "@/components/form/formCheckbox";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, BackupPlan } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { nameField, scheduleField } from "@/utils/validation";

const formSchema = v.pipe(
    v.object({
        accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
        name: nameField,
        schedule: scheduleField,
        retentionCount: v.pipe(
            v.string(),
            v.regex(/^\d+$/, "Enter a whole retention number."),
            v.transform(Number),
            v.integer("Enter a whole retention number."),
            v.minValue(1, "Keep at least one backup."),
            v.maxValue(100, "Keep no more than 100 backups."),
        ),
        includeFiles: v.boolean(),
        includeDatabases: v.boolean(),
    }),
    v.forward(
        v.partialCheck(
            [["includeFiles"], ["includeDatabases"]],
            ({ includeFiles, includeDatabases }) => includeFiles || includeDatabases,
            "Choose what to back up.",
        ),
        ["includeFiles"],
    ),
);

type FormInputValues = v.InferInput<typeof formSchema>;
type FormValues = v.InferOutput<typeof formSchema>;

type CreateBackupProps = {
    accounts: AccountListResponse;
};

const CreateBackup = ({ accounts }: CreateBackupProps) => {
    const [open, setOpen] = useState(false);

    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });

    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];

    const form = useForm<FormInputValues, unknown, FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            accountId: options[0]?.id ?? "",
            name: "Daily backup",
            schedule: "0 3 * * *",
            retentionCount: "7",
            includeFiles: true,
            includeDatabases: true,
        },
    });

    const accountId = useWatch({ control: form.control, name: "accountId" });
    const files = useWatch({ control: form.control, name: "includeFiles" });
    const databases = useWatch({ control: form.control, name: "includeDatabases" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    await api
                        .post("backup-plans", { json: { ...values, enabled: true } })
                        .json<BackupPlan>();

                    await Promise.all([
                        mutate("backup-plans"),
                        mutate("backup-runs"),
                        mutate("backup-artifacts"),
                        mutate("jobs"),
                    ]);

                    setOpen(false);
                    toast.success("Backup plan created");
                } catch (error) {
                    toast.danger("Backup action failed", {
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
                title="Create backup plan"
                description="Creates a scheduled or on-demand local backup plan."
                triggerLabel="Create backup plan"
                submitLabel="Create plan"
                pending={pending}
                size="lg"
                submitDisabled={!accountId || (!files && !databases)}
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
                    label="Schedule (blank for manual)"
                    maxLength={100}
                />
                <FormInput
                    name="retentionCount"
                    label="Retention"
                    type="number"
                    min="1"
                    max="100"
                    required
                />
                <div className="space-y-3 pt-1">
                    <FormCheckbox name="includeFiles" label="Files" />
                    <FormCheckbox name="includeDatabases" label="Databases" />
                </div>
            </FormModal>
        </Form>
    );
};

export default CreateBackup;
