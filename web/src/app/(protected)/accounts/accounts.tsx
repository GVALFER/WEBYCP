"use client";

import { UserRound } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { AccountListResponse, NodeListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import AccountActions from "./actions/accountActions";
import CreateAccount from "./actions/createAccount";

type AccountsProps = {
    accounts: AccountListResponse;
    nodes: NodeListResponse;
};

type Account = AccountListResponse["items"][number];

const Accounts = ({ accounts, nodes }: AccountsProps) => {
    const { dt } = useTimezone();

    const table = useTable(accounts.pagination);

    const { data } = useSWR<AccountListResponse>(`accounts${table.query}`, {
        fallbackData: table.isInitialQuery ? accounts : undefined,
    });

    const { data: nodesData } = useSWR<NodeListResponse>("nodes", {
        fallbackData: nodes,
    });

    const nodeId = nodesData?.items[0]?.id ?? "";

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
            render: (account) => <AccountActions account={account} />,
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
                <CreateAccount nodeId={nodeId} />
            </div>
            <Table table={table} columns={columns} data={data} />
        </section>
    );
};

export default Accounts;
