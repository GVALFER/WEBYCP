"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { toast } from "@heroui/react";
import { Pencil } from "lucide-react";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR, { useSWRConfig } from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormCheckbox } from "@/components/form/formCheckbox";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type { AccountListResponse, BackupPlan, ServiceDefaults } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { SERVICE_OPTIONS } from "@/utils/services";
import { nameField, scheduleField } from "@/utils/validation";

const formSchema = v.pipe(
    v.object({
        accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
        name: nameField,
        schedule: scheduleField,
        retentionCount: v.pipe(
            v.number("Enter a retention number."),
            v.integer("Enter a whole retention number."),
            v.minValue(1, "Keep at least one backup."),
            v.maxValue(100, "Keep no more than 100 backups."),
        ),
        includeFiles: v.boolean(),
        includeDatabases: v.boolean(),
        storageDriver: v.literal("local"),
        enabled: v.boolean(),
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

type Props = {
    accounts: AccountListResponse;
    driver: ServiceDefaults["backupDriver"];
    plan?: BackupPlan;
};

const PlanForm = ({ accounts, driver, plan }: Props) => {
    const [open, setOpen] = useState(false);

    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<AccountListResponse>(pageKey("accounts", { page: 1, size: 100 }), {
        fallbackData: accounts,
    });

    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];

    const defaults: FormInputValues = plan
        ? {
              accountId: plan.accountId,
              name: plan.name,
              schedule: plan.schedule,
              retentionCount: plan.retentionCount,
              includeFiles: plan.includeFiles,
              includeDatabases: plan.includeDatabases,
              storageDriver: plan.storageDriver,
              enabled: plan.enabled,
          }
        : {
              accountId: options[0]?.id ?? "",
              name: "Daily backup",
              schedule: "0 3 * * *",
              retentionCount: 7,
              includeFiles: true,
              includeDatabases: true,
              storageDriver: driver,
              enabled: true,
          };

    const form = useForm<FormInputValues, unknown, FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: defaults,
    });

    const changeOpen = (value: boolean) => {
        if (pending) return;
        if (value) form.reset(defaults);
        setOpen(value);
    };

    const accountId = useWatch({ control: form.control, name: "accountId" });
    const files = useWatch({ control: form.control, name: "includeFiles" });
    const databases = useWatch({ control: form.control, name: "includeDatabases" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    if (plan) {
                        await api.patch(`backup-plans/${encodeURIComponent(plan.id)}`, {
                            json: values,
                        }).json<BackupPlan>();
                    } else {
                        await api.post("backup-plans", { json: values }).json<BackupPlan>();
                    }

                    await mutate((key) =>
                        isPageKey(
                            key,
                            "backup-plans",
                            "backup-runs",
                            "backup-artifacts",
                            "accounts",
                            "jobs",
                        ),
                    );

                    setOpen(false);
                    toast.success(plan ? "Backup plan updated" : "Backup plan created");
                } catch (error) {
                    toast.danger("Backup action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [mutate, plan],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title={plan ? "Edit backup plan" : "Create backup plan"}
                description="Account backups stored on the account's server. Schedules run in UTC."
                triggerLabel={plan ? `Edit ${plan.name}` : "Create backup plan"}
                triggerIcon={plan ? <Pencil className="size-4" /> : undefined}
                triggerVariant={plan ? "secondary" : "primary"}
                submitLabel={plan ? "Save plan" : "Create plan"}
                pending={pending}
                size="lg"
                submitDisabled={!accountId || (!files && !databases)}
                onOpenChange={changeOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                {!plan && (
                    <FormSelect
                        name="accountId"
                        label="Hosting account"
                        options={options}
                        empty="No active accounts"
                        required
                    />
                )}
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
                <FormSelect
                    name="storageDriver"
                    label="Backup storage"
                    options={SERVICE_OPTIONS.backupDriver}
                    required
                />
                <div className="space-y-3 pt-1">
                    <FormCheckbox name="includeFiles" label="Files" />
                    <FormCheckbox name="includeDatabases" label="Databases" />
                    <FormCheckbox name="enabled" label="Enable schedule" />
                </div>
            </FormModal>
        </Form>
    );
};

export default PlanForm;
