"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { Button, toast } from "@heroui/react";
import { Plus } from "lucide-react";
import { useCallback, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormCheckbox } from "@/components/form/formCheckbox";
import { FormInput } from "@/components/form/formInput";
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
    const { mutate } = useSWRConfig();
    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];
    const [pending, startTransition] = useTransition();
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
            <form className="panel-card h-fit p-6" onSubmit={form.handleSubmit(handleSubmit)}>
                <h2 className="text-base font-semibold">Create backup plan</h2>
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
                    label="Schedule (blank for manual)"
                    maxLength={100}
                />
                <FormInput
                    className="mt-4"
                    name="retentionCount"
                    label="Retention"
                    type="number"
                    min="1"
                    max="100"
                    required
                />
                <FormCheckbox className="mt-5" name="includeFiles" label="Files" />
                <FormCheckbox className="mt-3" name="includeDatabases" label="Databases" />
                <Button
                    className="mt-6"
                    type="submit"
                    variant="primary"
                    fullWidth
                    isDisabled={!accountId || (!files && !databases) || pending}
                >
                    <Plus className="size-4" />
                    Create plan
                </Button>
            </form>
        </Form>
    );
};

export default CreateBackup;
