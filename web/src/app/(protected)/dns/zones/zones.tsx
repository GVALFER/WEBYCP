"use client";

import { Globe2 } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    DNSProviderListResponse,
    DNSSettings,
    DNSZoneListResponse,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreateZone from "./actions/createZone";
import ZoneActions from "./actions/zoneActions";

type Props = {
    accounts: AccountListResponse;
    providers: DNSProviderListResponse;
    settings: DNSSettings;
    zones: DNSZoneListResponse;
};

type Zone = DNSZoneListResponse["items"][number];

const Zones = ({ accounts, providers, settings, zones }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(zones.pagination);

    const { data, isLoading } = useSWR<DNSZoneListResponse>(`dns/zones${table.query}`, {
        fallbackData: table.isInitialQuery ? zones : undefined,
    });

    const accountNames = new Map(accounts.items.map((item) => [item.id, item.name]));
    const providerNames = new Map(providers.items.map((item) => [item.id, item.name]));

    const columns: TableColumn<Zone>[] = [
        {
            id: "zone",
            label: "Zone",
            isRowHeader: true,
            render: (zone) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <Globe2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <Link
                            className="font-medium hover:text-accent"
                            href={`/dns/zones/${encodeURIComponent(zone.id)}`}
                            prefetch={false}
                        >
                            {zone.name}
                        </Link>
                        <div className="mt-1 text-xs text-foreground-400">
                            {accountNames.get(zone.accountId) ?? zone.accountId}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "provider",
            label: "Provider",
            render: (zone) => providerNames.get(zone.providerId) ?? zone.providerId,
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (zone) => dt(zone.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (zone) => (
                <span
                    className={cn(
                        "inline-flex rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(zone.status),
                    )}
                >
                    {zone.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (zone) => <ZoneActions zone={zone} />,
        },
    ];

    const configured = Boolean(settings.primaryNameserver && settings.secondaryNameserver);

    return (
        <div className="space-y-5">
            {!configured && (
                <div className="rounded-xl border border-warning/30 bg-warning/10 px-5 py-4 text-sm text-warning-foreground">
                    Configure both default nameservers before creating a zone.{" "}
                    <Link
                        className="font-medium underline"
                        href="/dns/nameservers"
                        prefetch={false}
                    >
                        Open nameserver settings
                    </Link>
                </div>
            )}
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Authoritative zones</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            DNS zones are independent from website hostname bindings.
                        </div>
                    </div>
                    <CreateZone
                        accounts={accounts.items}
                        providers={providers.items}
                        configured={configured}
                    />
                </div>
                <Table table={table} columns={columns} data={data} pending={isLoading} />
            </section>
        </div>
    );
};

export default Zones;
