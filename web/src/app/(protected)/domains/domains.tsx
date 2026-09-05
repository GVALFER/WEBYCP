"use client";

import { Globe2 } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { WebsiteDomainListResponse, WebsiteListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import DomainActions from "./actions/domainActions";

type Props = {
    domains: WebsiteDomainListResponse;
    websites: WebsiteListResponse;
};

type Domain = WebsiteDomainListResponse["items"][number];

const Domains = ({ domains: initial, websites }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(initial.pagination);

    const { data, isLoading } = useSWR<WebsiteDomainListResponse>(
        `website-domains?kind=primary&${table.query.slice(1)}`,
        { fallbackData: table.isInitialQuery ? initial : undefined },
    );
    const { data: websiteData } = useSWR<WebsiteListResponse>(
        pageKey("websites", { page: 1, size: 100 }),
        { fallbackData: websites },
    );

    const names = new Map(websiteData?.items.map((item) => [item.id, item.name]));

    const columns: TableColumn<Domain>[] = [
        {
            id: "domain",
            label: "Domain",
            isRowHeader: true,
            render: (domain) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <Globe2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{domain.hostname}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {names.get(domain.websiteId) ?? domain.websiteId}
                        </div>
                    </div>
                </div>
            ),
        },
        { id: "type", label: "Type", render: () => "Primary" },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (domain) => dt(domain.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (domain) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(domain.status),
                    )}
                >
                    {domain.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (domain) => <DomainActions domain={domain} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="px-6 py-5">
                <h2 className="text-base font-semibold">Primary domains</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    The canonical hostname assigned to each website.
                </div>
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Domains;
