"use client";

import { Database as DatabaseIcon, UserRound } from "lucide-react";
import useSWR from "swr";
import type {
    AccountListResponse,
    DatabaseGrantListResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreateResources from "./actions/createResources";
import GrantActions from "./actions/grantActions";
import ResourceActions from "./actions/resourceActions";

type DatabasesProps = {
    accounts: AccountListResponse;
    databases: DatabaseListResponse;
    users: DatabaseUserListResponse;
    grants: DatabaseGrantListResponse;
};

const Databases = ({
    accounts,
    databases: initialDatabases,
    users: initialUsers,
    grants: initialGrants,
}: DatabasesProps) => {
    const { data: databases } = useSWR<DatabaseListResponse>("databases", {
        fallbackData: initialDatabases,
    });
    const { data: users } = useSWR<DatabaseUserListResponse>("database-users", {
        fallbackData: initialUsers,
    });
    const { data: grants } = useSWR<DatabaseGrantListResponse>("database-grants", {
        fallbackData: initialGrants,
    });

    return (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <div className="space-y-6">
                <ResourceList
                    title="MySQL databases"
                    empty="No databases yet."
                    items={databases?.items ?? []}
                    icon={DatabaseIcon}
                    kind="database"
                />
                <ResourceList
                    title="MySQL users"
                    empty="No database users yet."
                    items={users?.items ?? []}
                    icon={UserRound}
                    kind="user"
                />
                <section className="panel-card p-6">
                    <h2 className="text-base font-semibold">Grants</h2>
                    <div className="mt-4 space-y-2">
                        {grants?.items.length ? (
                            grants.items.map((grant) => {
                                const database = databases?.items.find(
                                    (item) => item.id === grant.databaseId,
                                );
                                const user = users?.items.find(
                                    (item) => item.id === grant.databaseUserId,
                                );

                                return (
                                    <div
                                        key={`${grant.databaseId}:${grant.databaseUserId}`}
                                        className="flex items-center justify-between rounded-xl border border-border/70 bg-surface-secondary/60 px-4 py-3 text-sm"
                                    >
                                        <div>
                                            <span className="font-medium">{user?.name}</span>
                                            <span className="mx-2 text-foreground-400">→</span>
                                            {database?.name}
                                        </div>
                                        <GrantActions
                                            grant={grant}
                                            database={database?.name ?? ""}
                                            user={user?.name ?? ""}
                                        />
                                    </div>
                                );
                            })
                        ) : (
                            <div className="py-6 text-center text-sm text-foreground-400">
                                No grants yet.
                            </div>
                        )}
                    </div>
                </section>
            </div>

            <CreateResources
                accounts={accounts}
                databases={initialDatabases}
                users={initialUsers}
            />
        </div>
    );
};

export default Databases;

type Resource =
    | DatabaseListResponse["items"][number]
    | DatabaseUserListResponse["items"][number];

type ResourceListProps = {
    title: string;
    empty: string;
    items: Resource[];
    icon: typeof DatabaseIcon;
    kind: "database" | "user";
};

const ResourceList = ({ title, empty, items, icon: Icon, kind }: ResourceListProps) => (
    <section className="panel-card overflow-hidden">
        <div className="border-b border-divider px-6 py-5">
            <h2 className="text-base font-semibold">{title}</h2>
        </div>
        <div className="divide-y divide-divider">
            {items.length ? (
                items.map((item) => (
                    <div
                        key={item.id}
                        className="flex items-center justify-between gap-4 px-6 py-4"
                    >
                        <div className="flex min-w-0 items-center gap-3">
                            <Icon className="size-4 shrink-0 text-foreground-400" />
                            <div className="min-w-0">
                                <div className="truncate font-medium">{item.name}</div>
                                <div className="truncate font-mono text-xs text-foreground-400">
                                    {item.systemName}
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-2">
                            <span
                                className={cn(
                                    "rounded-full px-2 py-1 text-xs capitalize",
                                    statusClass(item.status),
                                )}
                            >
                                {item.status}
                            </span>
                            <ResourceActions kind={kind} resource={item} />
                        </div>
                    </div>
                ))
            ) : (
                <div className="px-6 py-10 text-center text-sm text-foreground-400">{empty}</div>
            )}
        </div>
    </section>
);
