"use client";

import { UserRound } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { AccountListResponse, NodeListResponse, PackageListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import AccountActions from "./actions/accountActions";
import CreateAccount from "./actions/createAccount";

type AccountsProps = {
    accounts: AccountListResponse;
    nodes: NodeListResponse;
    packages: PackageListResponse;
};

type Account = AccountListResponse["items"][number];

const Accounts = ({ accounts, nodes, packages }: AccountsProps) => {
    const { dt } = useTimezone();
    const table = useTable(accounts.pagination);

    const { data, isLoading } = useSWR<AccountListResponse>(`accounts${table.query}`, {
        fallbackData: table.isInitialQuery ? accounts : undefined,
    });
    const { data: nodesData } = useSWR<NodeListResponse>("nodes", {
        fallbackData: nodes,
    });
    const { data: packagesData } = useSWR<PackageListResponse>("packages?page=1&size=100", {
        fallbackData: packages,
    });

    const nodeId = nodesData?.items[0]?.id ?? "";
    const packageItems = packagesData?.items ?? [];

    const columns: TableColumn<Account>[] = [
        {
            id: "account",
            label: "Account",
            isRowHeader: true,
            render: (account) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <UserRound className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{account.name}</div>
                        <div className="mt-1 font-mono text-xs text-foreground-400">
                            {account.systemUser}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "package",
            label: "Package",
            render: ({ package: value }) => (
                <div>
                    <div className="font-medium">{value.name}</div>
                    <div className="mt-1 text-xs text-foreground-400">
                        {value.accountCount} assigned
                    </div>
                </div>
            ),
        },
        {
            id: "usage",
            label: "Usage",
            cellClassName: "min-w-72",
            render: ({ package: value, usage }) => (
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-foreground-500">
                    <Usage label="Websites" used={usage.websites} limit={value.limits.websites} />
                    <Usage label="Domains" used={usage.domains} limit={value.limits.domains} />
                    <Usage label="Aliases" used={usage.aliases} limit={value.limits.aliases} />
                    <Usage label="FTP" used={usage.ftpAccounts} limit={value.limits.ftpAccounts} />
                    <Usage
                        label="Databases"
                        used={usage.databases}
                        limit={value.limits.databases}
                    />
                    <Usage
                        label="DB users"
                        used={usage.databaseUsers}
                        limit={value.limits.databaseUsers}
                    />
                    <Usage
                        label="Tasks"
                        used={usage.scheduledTasks}
                        limit={value.limits.scheduledTasks}
                    />
                    <Usage
                        label="Backup plans"
                        used={usage.backupPlans}
                        limit={value.limits.backupPlans}
                    />
                </div>
            ),
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (account) => dt(account.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (account) => (
                <span
                    className={cn(
                        "inline-flex rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(account.status),
                    )}
                >
                    {account.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (account) => (
                <AccountActions
                    account={account}
                    packageId={account.package.id}
                    packages={packageItems}
                />
            ),
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Hosting accounts</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Isolated Linux identities owned by panel users.
                    </div>
                </div>
                <CreateAccount nodeId={nodeId} packages={packageItems} />
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

type UsageProps = {
    label: string;
    used: number;
    limit: number;
};

const Usage = ({ label, used, limit }: UsageProps) => (
    <div className={cn("flex justify-between gap-2", used >= limit && "text-warning")}>
        <span>{label}</span>
        <span className="font-medium tabular-nums text-foreground">
            {used}/{limit}
        </span>
    </div>
);

export default Accounts;
