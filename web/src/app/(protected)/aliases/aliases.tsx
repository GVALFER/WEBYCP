"use client";

import { CornerDownRight } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { WebsiteDomainListResponse, WebsiteListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import AliasActions from "./actions/aliasActions";
import CreateAlias from "./actions/createAlias";

type Props = {
    aliases: WebsiteDomainListResponse;
    websites: WebsiteListResponse;
};

type Alias = WebsiteDomainListResponse["items"][number];

const Aliases = ({ aliases: initial, websites }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(initial.pagination);

    const { data, isLoading } = useSWR<WebsiteDomainListResponse>(
        `website-domains?kind=alias&${table.query.slice(1)}`,
        { fallbackData: table.isInitialQuery ? initial : undefined },
    );
    const { data: websiteData } = useSWR<WebsiteListResponse>(
        pageKey("websites", { page: 1, size: 100 }),
        { fallbackData: websites },
    );

    const names = new Map(websiteData?.items.map((item) => [item.id, item.name]));

    const columns: TableColumn<Alias>[] = [
        {
            id: "alias",
            label: "Alias",
            isRowHeader: true,
            render: (alias) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <CornerDownRight className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{alias.hostname}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {names.get(alias.websiteId) ?? alias.websiteId}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (alias) => dt(alias.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (alias) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(alias.status),
                    )}
                >
                    {alias.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (alias) => <AliasActions alias={alias} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Aliases</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Additional hostnames serving an existing website.
                    </div>
                </div>
                <CreateAlias websites={websiteData ?? websites} />
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Aliases;
