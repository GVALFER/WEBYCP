"use client";

import { useState } from "react";
import useSWR from "swr";
import type {
    AccountListResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { SelectField } from "@/components/SelectField";
import CreateDatabase from "./createDatabase";
import CreateDatabaseUser from "./createDatabaseUser";
import CreateGrant from "./createGrant";

type CreateResourcesProps = {
    accounts: AccountListResponse;
    databases: DatabaseListResponse;
    users: DatabaseUserListResponse;
};

const CreateResources = ({ accounts, databases, users }: CreateResourcesProps) => {
    const [accountId, setAccountId] = useState("");
    const [password, setPassword] = useState("");
    const { data: accountsData } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const { data: databaseData } = useSWR<DatabaseListResponse>("databases", {
        fallbackData: databases,
    });
    const { data: userData } = useSWR<DatabaseUserListResponse>("database-users", {
        fallbackData: users,
    });
    const options =
        accountsData?.items.filter(
            (account) => account.status === "active" && account.enabled,
        ) ?? [];
    const selected = accountId || options[0]?.id || "";
    const accountDatabases =
        databaseData?.items.filter(
            (item) => item.accountId === selected && item.status === "active",
        ) ?? [];
    const accountUsers =
        userData?.items.filter(
            (item) => item.accountId === selected && item.status === "active",
        ) ?? [];

    return (
        <aside className="space-y-6">
            {password && (
                <div className="rounded-2xl border border-warning/30 bg-warning/10 p-5">
                    <div className="font-medium">Save this password now</div>
                    <div className="mt-3 select-all break-all rounded-xl bg-background/70 px-4 py-3 font-mono text-sm">
                        {password}
                    </div>
                    <div className="mt-2 text-xs text-foreground-500">
                        It will not be shown again.
                    </div>
                </div>
            )}
            <section className="panel-card p-6">
                <h2 className="text-base font-semibold">Create resources</h2>
                <SelectField
                    className="mt-5"
                    label="Hosting account"
                    value={selected}
                    options={options}
                    onChange={setAccountId}
                />
                <CreateDatabase accountId={selected} />
                <CreateDatabaseUser accountId={selected} onPassword={setPassword} />
            </section>
            <CreateGrant databases={accountDatabases} users={accountUsers} />
        </aside>
    );
};

export default CreateResources;
