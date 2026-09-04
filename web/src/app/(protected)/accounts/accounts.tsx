"use client";

import { UserRound } from "lucide-react";
import useSWR from "swr";
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

const Accounts = ({ accounts, nodes }: AccountsProps) => {
    const { dt } = useTimezone();

    const { data: nodesData } = useSWR<NodeListResponse>("nodes", {
        fallbackData: nodes,
    });
    const { data } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });

    const nodeId = nodesData?.items[0]?.id ?? "";

    return (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <section className="panel-card overflow-hidden">
                <div className="border-b border-divider px-6 py-5">
                    <h2 className="text-base font-semibold">Hosting accounts</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Isolated Linux identities owned by panel users.
                    </div>
                </div>
                <div className="divide-y divide-divider">
                    {data?.items.length ? (
                        data.items.map((account) => (
                            <div
                                key={account.id}
                                className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                            >
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
                                <div className="flex items-center gap-4">
                                    <div className="hidden text-xs text-foreground-400 sm:block">
                                        {dt(account.createdAt)}
                                    </div>
                                    <span
                                        className={cn(
                                            "rounded-full px-2.5 py-1 text-xs capitalize",
                                            statusClass(account.status),
                                        )}
                                    >
                                        {account.status}
                                    </span>
                                    <AccountActions account={account} />
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className="px-6 py-12 text-center text-sm text-foreground-400">
                            No hosting accounts yet.
                        </div>
                    )}
                </div>
            </section>

            <aside className="panel-card h-fit p-6">
                <h2 className="text-base font-semibold">Create account</h2>
                <div className="mt-1 text-sm leading-6 text-foreground-500">
                    This queues an isolated Linux user on the selected node.
                </div>
                <CreateAccount nodeId={nodeId} />
            </aside>
        </div>
    );
};

export default Accounts;
