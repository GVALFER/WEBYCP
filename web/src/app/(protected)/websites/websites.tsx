"use client";

import { Code2 } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    ServiceSettings,
    WebsiteDomainListResponse,
    WebsiteListResponse,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import CreateWebsite from "./actions/createWebsite";
import WebsiteActions from "./actions/websiteActions";

type Props = {
    accounts: AccountListResponse;
    websites: WebsiteListResponse;
    domains: WebsiteDomainListResponse;
    settings: ServiceSettings;
};

type Website = WebsiteListResponse["items"][number];

const Websites = ({ accounts, websites: initial, domains, settings: initialSettings }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(initial.pagination);

    const { data } = useSWR<WebsiteListResponse>(`websites${table.query}`, {
        fallbackData: table.isInitialQuery ? initial : undefined,
    });
    const { data: accountData } = useSWR<AccountListResponse>(
        pageKey("accounts", { page: 1, size: 100 }),
        { fallbackData: accounts },
    );
    const { data: domainData } = useSWR<WebsiteDomainListResponse>(
        "website-domains?kind=primary&page=1&size=100",
        { fallbackData: domains },
    );
    const { data: settings = initialSettings } = useSWR<ServiceSettings>("service-settings", {
        fallbackData: initialSettings,
    });

    const accountNames = new Map(accountData?.items.map((item) => [item.id, item.name]));
    const primaryDomains = new Map(
        domainData?.items.map((item) => [item.websiteId, item.hostname]),
    );

    const columns: TableColumn<Website>[] = [
        {
            id: "website",
            label: "Website",
            isRowHeader: true,
            render: (website) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <Code2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{website.name}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {primaryDomains.get(website.id) ?? "No primary domain"}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "account",
            label: "Account",
            render: (website) => accountNames.get(website.accountId) ?? website.accountId,
        },
        {
            id: "stack",
            label: "Stack",
            render: (website) =>
                `${website.webDriver} · ${website.runtimeDriver} ${website.runtimeVersion}`,
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (website) => dt(website.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (website) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(website.status),
                    )}
                >
                    {website.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (website) => <WebsiteActions website={website} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Websites</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Sites, document roots and runtime stacks.
                    </div>
                </div>
                <CreateWebsite accounts={accountData ?? accounts} defaults={settings.defaults} />
            </div>
            <Table table={table} columns={columns} data={data} />
        </section>
    );
};

export default Websites;
