"use client";

import { FileKey2 } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { DNSRecordListResponse, DNSZone } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreateRecord from "./actions/createRecord";
import RecordActions from "./actions/recordActions";

type Props = {
    defaultTtl: number;
    records: DNSRecordListResponse;
    zone: DNSZone;
};

type Record = DNSRecordListResponse["items"][number];

const Records = ({ defaultTtl, records, zone }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(records.pagination);
    const path = `dns/zones/${encodeURIComponent(zone.id)}/records`;
    const { data } = useSWR<DNSRecordListResponse>(`${path}${table.query}`, {
        fallbackData: table.isInitialQuery ? records : undefined,
    });

    const columns: TableColumn<Record>[] = [
        {
            id: "record",
            label: "Record",
            isRowHeader: true,
            render: (record) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <FileKey2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{record.name}</div>
                        <div className="mt-1 font-mono text-xs text-foreground-400">
                            {record.type}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "content",
            label: "Content",
            cellClassName: "max-w-md",
            render: (record) => (
                <div className="break-all font-mono text-sm">
                    {record.type === "MX" ? `${record.priority} ${record.content}` : record.content}
                </div>
            ),
        },
        {
            id: "ttl",
            label: "TTL",
            cellClassName: "whitespace-nowrap tabular-nums text-foreground-500",
            render: (record) => `${record.ttl}s`,
        },
        {
            id: "updated",
            label: "Updated",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (record) => dt(record.updatedAt),
        },
        {
            id: "status",
            label: "Status",
            render: (record) => (
                <span
                    className={cn(
                        "inline-flex rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(record.status),
                    )}
                >
                    {record.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (record) => <RecordActions record={record} zoneName={zone.name} />,
        },
    ];

    return (
        <div className="space-y-5">
            <div className="text-sm text-foreground-500">
                <Link className="hover:text-accent" href="/dns/zones" prefetch={false}>
                    DNS Zones
                </Link>
                <span className="mx-2">/</span>
                <span className="text-foreground">{zone.name}</span>
            </div>
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">DNS records</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            A, AAAA, CNAME, MX and TXT records for {zone.name}.
                        </div>
                    </div>
                    <CreateRecord zone={zone} defaultTtl={defaultTtl} />
                </div>
                <Table table={table} columns={columns} data={data} />
            </section>
        </div>
    );
};

export default Records;
