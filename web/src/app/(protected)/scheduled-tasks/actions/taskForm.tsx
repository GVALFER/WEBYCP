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
import type {
    AccountListResponse,
    ScheduledTaskResponse,
    ScheduledTask,
    ServiceDefaults,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey, pageKey } from "@/utils/pagination";
import { SERVICE_OPTIONS } from "@/utils/services";
import { commandField, nameField, scheduleField } from "@/utils/validation";

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    name: nameField,
    schedule: v.pipe(scheduleField, v.nonEmpty("Enter a schedule.")),
    command: commandField,
    schedulerDriver: v.literal("crontab"),
    kind: v.literal("command"),
    enabled: v.boolean(),
});

type FormValues = v.InferOutput<typeof formSchema>;

type TaskFormProps = {
    accounts: AccountListResponse;
    driver: ServiceDefaults["schedulerDriver"];
    task?: ScheduledTask;
};

const TaskForm = ({ accounts, driver, task }: TaskFormProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { mutate } = useSWRConfig();

    const { data } = useSWR<AccountListResponse>(pageKey("accounts", { page: 1, size: 100 }), {
        fallbackData: accounts,
    });

    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];

    const defaults: FormValues = task
        ? {
              accountId: task.accountId,
              name: task.name,
              schedule: task.schedule,
              command: task.command,
              schedulerDriver: task.schedulerDriver,
              kind: task.kind,
              enabled: task.enabled,
          }
        : {
              accountId: options[0]?.id ?? "",
              name: "",
              schedule: "0 * * * *",
              command: "",
              schedulerDriver: driver,
              kind: "command",
              enabled: true,
          };

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: defaults,
    });

    const changeOpen = (value: boolean) => {
        if (pending) return;
        if (value) form.reset(defaults);
        setOpen(value);
    };

    const accountId = useWatch({ control: form.control, name: "accountId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                try {
                    if (task) {
                        await api
                            .patch(`scheduled-tasks/${encodeURIComponent(task.id)}`, {
                                json: values,
                            })
                            .json<ScheduledTaskResponse>();
                    } else {
                        await api
                            .post("scheduled-tasks", { json: values })
                            .json<ScheduledTaskResponse>();
                    }
                    await mutate((key) =>
                        isPageKey(key, "scheduled-tasks", "accounts", "jobs", "audit-events"),
                    );
                    setOpen(false);
                    toast.success(
                        task ? "Scheduled task update queued" : "Scheduled task creation queued",
                    );
                } catch (error) {
                    toast.danger("Action failed", {
                        description: await errorMessage(error),
                    });
                }
            });
        },
        [mutate, task],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title={task ? "Edit scheduled task" : "Add scheduled task"}
                description="Runs a command as the hosting account from its home directory."
                triggerLabel={task ? `Edit ${task.name}` : "Add scheduled task"}
                triggerIcon={task ? <Pencil className="size-4" /> : undefined}
                triggerVariant={task ? "secondary" : "primary"}
                submitLabel={task ? "Save task" : "Add scheduled task"}
                pending={pending}
                submitDisabled={!accountId}
                onOpenChange={changeOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                {!task && (
                    <FormSelect
                        name="accountId"
                        label="Hosting account"
                        options={options}
                        empty="No active accounts"
                        required
                    />
                )}
                <FormInput name="name" label="Name" maxLength={80} required />
                <FormSelect
                    name="kind"
                    label="Task type"
                    options={[{ id: "command", name: "Command" }]}
                    required
                />
                <FormInput
                    inputClassName="font-mono"
                    name="schedule"
                    label="Schedule"
                    maxLength={100}
                    required
                />
                <FormSelect
                    name="schedulerDriver"
                    label="Scheduler"
                    options={SERVICE_OPTIONS.schedulerDriver}
                    required
                />
                <FormInput
                    inputClassName="font-mono"
                    name="command"
                    label="Command"
                    maxLength={1_000}
                    required
                />
                <FormCheckbox name="enabled" label="Enable task" />
            </FormModal>
        </Form>
    );
};

export default TaskForm;
