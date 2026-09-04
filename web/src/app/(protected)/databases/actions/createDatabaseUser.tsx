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
    DatabaseUserJobResponse,
    ServiceDefaults,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { pageKey } from "@/utils/pagination";
import { SERVICE_OPTIONS } from "@/utils/services";
import { dbNameField } from "@/utils/validation";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateDatabaseUserProps = {
    accounts: AccountListResponse;
    driver: ServiceDefaults["databaseDriver"];
};

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    name: dbNameField,
    driver: v.literal("mysql"),
});
type FormValues = v.InferOutput<typeof formSchema>;

const CreateDatabaseUser = ({ accounts, driver }: CreateDatabaseUserProps) => {
    const [open, setOpen] = useState(false);
    const [password, setPassword] = useState("");

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

    const handleOpenChange = useCallback((value: boolean) => {
        setOpen(value);
        if (!value) setPassword("");
    }, []);

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                const response = await run(
                    () =>
                        api
                            .post("database-users", { json: values })
                            .json<DatabaseUserJobResponse>(),
                    "Database user queued for creation",
                    true,
                );
                if (response) {
                    form.reset({ accountId: values.accountId, name: "", driver: values.driver });
                    setPassword(response.password ?? "");
                }
            });
        },
        [form, run],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Create database user"
                description="Creates isolated MySQL credentials for a hosting account."
                triggerLabel="Create database user"
                submitLabel={password ? "Create another user" : "Create user"}
                pending={pending}
                submitDisabled={!accountId}
                onOpenChange={handleOpenChange}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                {password && (
                    <div className="rounded-xl border border-warning/30 bg-warning/10 p-4">
                        <div className="font-medium">Save this password now</div>
                        <div className="mt-3 select-all break-all rounded-lg bg-background/70 px-3 py-2 font-mono text-sm">
                            {password}
                        </div>
                        <div className="mt-2 text-xs text-foreground-500">
                            It will not be shown again.
                        </div>
                    </div>
                )}
                <FormSelect
                    name="accountId"
                    label="Hosting account"
                    options={options}
                    empty="No active accounts"
                    required
                />
                <FormInput name="name" label="User name" maxLength={32} required />
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

export default CreateDatabaseUser;
