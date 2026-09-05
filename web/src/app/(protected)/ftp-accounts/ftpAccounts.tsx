"use client";

import { FolderLock } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { AccountListResponse, FTPAccount, FTPAccountListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import FTPForm from "./actions/ftpForm";
import FTPActions from "./actions/ftpActions";

type Props = {
    accounts: AccountListResponse;
    ftp: FTPAccountListResponse;
};

const FTPAccounts = ({ accounts, ftp }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(ftp.pagination);
    const { data, isLoading } = useSWR<FTPAccountListResponse>(`ftp-accounts${table.query}`, {
        fallbackData: table.isInitialQuery ? ftp : undefined,
    });
    const { data: accountData } = useSWR<AccountListResponse>(pageKey("accounts", { page: 1, size: 100 }), {
        fallbackData: accounts,
    });

    const columns: TableColumn<FTPAccount>[] = [
        {
            id: "username",
            label: "Username",
            isRowHeader: true,
            render: (item) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <FolderLock className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{item.username}</div>
                        <div className="mt-1 text-xs text-foreground-400">{item.accountName}</div>
                    </div>
                </div>
            ),
        },
        {
            id: "home",
            label: "Directory",
            cellClassName: "font-mono text-xs text-foreground-500",
            render: (item) => item.home,
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (item) => dt(item.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (item) => {
                const inactive = item.status === "active" && item.accountStatus !== "active";
                return (
                    <div>
                        <span className={cn(
                            "rounded-full px-2.5 py-1 text-xs capitalize",
                            statusClass(inactive ? "disabled" : item.status),
                        )}>
                            {inactive ? "Account inactive" : item.status}
                        </span>
                        {item.deleting && (
                            <div className="mt-2 text-xs text-foreground-400">Awaiting removal</div>
                        )}
                    </div>
                );
            },
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (item) => (
                <div className="flex items-center gap-2">
                    {!item.deleting && item.status !== "pending" && <FTPForm item={item} />}
                    <FTPActions item={item} />
                </div>
            ),
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">FTP accounts</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Explicit FTPS on port 21. Each login is restricted to its hosting account home.
                    </div>
                </div>
                <FTPForm accounts={accountData?.items} />
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default FTPAccounts;
