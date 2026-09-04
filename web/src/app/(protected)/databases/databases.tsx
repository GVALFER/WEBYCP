"use client";

import { Database as DatabaseIcon, UserRound } from "lucide-react";
import type { ReactNode } from "react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable, type TableState } from "@/components/table/useTable";
import type {
    AccountListResponse,
    DatabaseGrantListResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import CreateDatabase from "./actions/createDatabase";
import CreateDatabaseUser from "./actions/createDatabaseUser";
import CreateGrant from "./actions/createGrant";
import GrantActions from "./actions/grantActions";
import ResourceActions from "./actions/resourceActions";

type DatabasesProps = {
    accounts: AccountListResponse;
    databases: DatabaseListResponse;
    users: DatabaseUserListResponse;
    grants: DatabaseGrantListResponse;
    databaseOptions: DatabaseListResponse;
    userOptions: DatabaseUserListResponse;
};

type Resource = DatabaseListResponse["items"][number] | DatabaseUserListResponse["items"][number];
type Grant = DatabaseGrantListResponse["items"][number];

const Databases = ({
    accounts,
    databases: initialDatabases,
    users: initialUsers,
    grants: initialGrants,
    databaseOptions,
    userOptions,
}: DatabasesProps) => {
    const databasesTable = useTable(initialDatabases.pagination, "databases");
    const usersTable = useTable(initialUsers.pagination, "users");
    const grantsTable = useTable(initialGrants.pagination, "grants");

    const { data: databases } = useSWR<DatabaseListResponse>(
        `databases${databasesTable.query}`,
        { fallbackData: databasesTable.isInitialQuery ? initialDatabases : undefined },
    );
    const { data: users } = useSWR<DatabaseUserListResponse>(
        `database-users${usersTable.query}`,
        { fallbackData: usersTable.isInitialQuery ? initialUsers : undefined },
    );
    const { data: grants } = useSWR<DatabaseGrantListResponse>(
        `database-grants${grantsTable.query}`,
        { fallbackData: grantsTable.isInitialQuery ? initialGrants : undefined },
    );
    const { data: allDatabases } = useSWR<DatabaseListResponse>(
        pageKey("databases", { page: 1, size: 100 }),
        { fallbackData: databaseOptions },
    );
    const { data: allUsers } = useSWR<DatabaseUserListResponse>(
        pageKey("database-users", { page: 1, size: 100 }),
        { fallbackData: userOptions },
    );

    const databaseNames = new Map(allDatabases?.items.map((item) => [item.id, item.name]));
    const userNames = new Map(allUsers?.items.map((item) => [item.id, item.name]));

    const grantColumns: TableColumn<Grant>[] = [
        {
            id: "user",
            label: "Database user",
            isRowHeader: true,
            render: (grant) => (
                <div>
                    <div className="font-medium">
                        {userNames.get(grant.databaseUserId) ?? grant.databaseUserId}
                    </div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">
                        {grant.databaseUserId}
                    </div>
                </div>
            ),
        },
        {
            id: "database",
            label: "Database",
            render: (grant) =>
                databaseNames.get(grant.databaseId) ?? grant.databaseId,
        },
        {
            id: "status",
            label: "Status",
            render: (grant) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(grant.status),
                    )}
                >
                    {grant.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (grant) => (
                <GrantActions
                    grant={grant}
                    database={databaseNames.get(grant.databaseId) ?? ""}
                    user={userNames.get(grant.databaseUserId) ?? ""}
                />
            ),
        },
    ];

    return (
        <div className="space-y-6">
            <ResourceList
                title="MySQL databases"
                data={databases}
                table={databasesTable}
                icon={DatabaseIcon}
                kind="database"
                action={<CreateDatabase accounts={accounts} />}
            />
            <ResourceList
                title="MySQL users"
                data={users}
                table={usersTable}
                icon={UserRound}
                kind="user"
                action={<CreateDatabaseUser accounts={accounts} />}
            />
            <section className="panel-card overflow-hidden">
                <div className="flex items-center justify-between gap-4 px-6 py-5">
                    <h2 className="text-base font-semibold">Grants</h2>
                    <CreateGrant
                        accounts={accounts}
                        databases={allDatabases ?? databaseOptions}
                        users={allUsers ?? userOptions}
                    />
                </div>
                <Table
                    table={grantsTable}
                    columns={grantColumns}
                    data={grants}
                    getKey={(grant) => `${grant.databaseId}:${grant.databaseUserId}`}
                />
            </section>
        </div>
    );
};

export default Databases;

type ResourceListProps = {
    action: ReactNode;
    title: string;
    data?: DatabaseListResponse | DatabaseUserListResponse;
    table: TableState;
    icon: typeof DatabaseIcon;
    kind: "database" | "user";
};

const ResourceList = ({ action, title, data, table, icon: Icon, kind }: ResourceListProps) => {
    const columns: TableColumn<Resource>[] = [
        {
            id: "resource",
            label: kind === "database" ? "Database" : "Database user",
            isRowHeader: true,
            render: (item) => (
                <div className="flex min-w-0 items-center gap-3">
                    <Icon className="size-4 shrink-0 text-foreground-400" aria-hidden="true" />
                    <div className="min-w-0">
                        <div className="truncate font-medium">{item.name}</div>
                        <div className="truncate font-mono text-xs text-foreground-400">
                            {item.systemName}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "status",
            label: "Status",
            render: (item) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(item.status),
                    )}
                >
                    {item.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (item) => <ResourceActions kind={kind} resource={item} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-center justify-between gap-4 px-6 py-5">
                <h2 className="text-base font-semibold">{title}</h2>
                {action}
            </div>
            <Table table={table} columns={columns} data={data} />
        </section>
    );
};
