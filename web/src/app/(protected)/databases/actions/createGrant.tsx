"use client";

import { valibotResolver } from "@hookform/resolvers/valibot";
import { useCallback, useEffect, useMemo, useState, useTransition } from "react";
import { useForm, useWatch } from "react-hook-form";
import useSWR from "swr";
import * as v from "valibot";
import { Form } from "@/components/form/form";
import { FormModal } from "@/components/form/formModal";
import { FormSelect } from "@/components/form/formSelect";
import type {
    AccountListResponse,
    DatabaseGrantJobResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { useDatabaseAction } from "./useDatabaseAction";

type CreateGrantProps = {
    accounts: AccountListResponse;
    databases: DatabaseListResponse;
    users: DatabaseUserListResponse;
};

const formSchema = v.object({
    accountId: v.pipe(v.string(), v.nonEmpty("Choose a hosting account.")),
    databaseId: v.pipe(v.string(), v.nonEmpty("Choose a database.")),
    userId: v.pipe(v.string(), v.nonEmpty("Choose a database user.")),
});

type FormValues = v.InferOutput<typeof formSchema>;

const CreateGrant = ({
    accounts,
    databases: initialDatabases,
    users: initialUsers,
}: CreateGrantProps) => {
    const [open, setOpen] = useState(false);
    const [pending, startTransition] = useTransition();
    const { run } = useDatabaseAction();

    const { data: accountsData } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const { data: databases } = useSWR<DatabaseListResponse>("databases", {
        fallbackData: initialDatabases,
    });
    const { data: users } = useSWR<DatabaseUserListResponse>("database-users", {
        fallbackData: initialUsers,
    });

    const accountsOptions = useMemo(
        () =>
            accountsData?.items.filter(
                (account) => account.status === "active" && account.enabled,
            ) ?? [],
        [accountsData?.items],
    );

    const initialAccountId = accountsOptions[0]?.id ?? "";

    const initialDatabaseId =
        databases?.items.find(
            (item) => item.accountId === initialAccountId && item.status === "active",
        )?.id ?? "";

    const initialUserId =
        users?.items.find((item) => item.accountId === initialAccountId && item.status === "active")
            ?.id ?? "";

    const form = useForm<FormValues>({
        resolver: valibotResolver(formSchema),
        defaultValues: {
            accountId: initialAccountId,
            databaseId: initialDatabaseId,
            userId: initialUserId,
        },
    });

    const accountId = useWatch({ control: form.control, name: "accountId" });
    const databaseOptions = useMemo(
        () =>
            databases?.items.filter(
                (item) => item.accountId === accountId && item.status === "active",
            ) ?? [],
        [accountId, databases?.items],
    );

    const userOptions = useMemo(
        () =>
            users?.items.filter(
                (item) => item.accountId === accountId && item.status === "active",
            ) ?? [],
        [accountId, users?.items],
    );

    const databaseId = useWatch({ control: form.control, name: "databaseId" });
    const userId = useWatch({ control: form.control, name: "userId" });

    useEffect(() => {
        if (!accountsOptions.some((item) => item.id === form.getValues("accountId"))) {
            form.setValue("accountId", accountsOptions[0]?.id ?? "");
        }
        if (!databaseOptions.some((item) => item.id === form.getValues("databaseId"))) {
            form.setValue("databaseId", databaseOptions[0]?.id ?? "");
        }
        if (!userOptions.some((item) => item.id === form.getValues("userId"))) {
            form.setValue("userId", userOptions[0]?.id ?? "");
        }
    }, [accountsOptions, databaseOptions, form, userOptions]);

    const handleSubmit = useCallback(
        (values: FormValues) => {
            startTransition(async () => {
                const path = `databases/${encodeURIComponent(values.databaseId)}/users/${encodeURIComponent(values.userId)}`;
                const response = await run(
                    () => api.put(path).json<DatabaseGrantJobResponse>(),
                    "Access granted",
                );
                if (response) setOpen(false);
            });
        },
        [run],
    );

    return (
        <Form {...form}>
            <FormModal
                open={open}
                title="Grant database access"
                description="Allows a database user to access a database in the same account."
                triggerLabel="Grant database access"
                submitLabel="Grant access"
                pending={pending}
                submitDisabled={!accountId || !databaseId || !userId}
                onOpenChange={setOpen}
                onSubmit={form.handleSubmit(handleSubmit)}
            >
                <FormSelect
                    name="accountId"
                    label="Hosting account"
                    options={accountsOptions}
                    empty="No active accounts"
                    required
                />
                <FormSelect
                    name="databaseId"
                    label="Database"
                    options={databaseOptions}
                    empty="No active databases"
                    required
                />
                <FormSelect
                    name="userId"
                    label="User"
                    options={userOptions}
                    empty="No active users"
                    required
                />
            </FormModal>
        </Form>
    );
};

export default CreateGrant;
