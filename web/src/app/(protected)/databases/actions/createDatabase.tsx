"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { useCallback, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormInput } from "@/components/form/formInput";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type {
    AccountListResponse,
    DatabaseJobResponse,
    ServiceDefaults,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { pageKey } from "@/utils/pagination";
import { SERVICE_OPTIONS } from "@/utils/services";
import { dbNameField } from "@/utils/validation";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateDatabaseProps = {
    accounts: AccountListResponse;
    driver: ServiceDefaults["databaseDriver"];
};

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    name: dbNameField,
    driver: v.literal("mysql"),
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateDatabase = ({ accounts, driver }: CreateDatabaseProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { run } = useDatabaseAction();

    const { data } = useSWR<AccountListResponse>(pageKey("accounts", { page: 1, size: 100 }), {
        fallbackData: accounts,
    });

    const options =
        data?.items.filter((account) => account.status === "active" && account.enabled) ?? [];

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: { accountId: options[0]?.id ?? "", name: "", driver },
    });

    const accountId = useWatch({ control: form.control, name: "accountId" });

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                const response = await run(
                    () => api.post("databases", { json: values }).json<DatabaseJobResponse>(),
                    "Database queued for creation",
                    true,
                );
                if (response) {
                    form.reset({ accountId: values.accountId, name: "", driver: values.driver });
                    setOpen(false);
                }
            });
        },
        [form, run],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Create database"
                description="Creates a MySQL database for a hosting account."
                triggerLabel="Create database"
                submitLabel="Create database"
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
                <FormInput name="name" label="Database name" maxLength={32} required />
                <FormSelect
                    name="driver"
                    label="Database service"
                    options={SERVICE_OPTIONS.databaseDriver}
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateDatabase;
