"use client";

import Link from "next/link";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { AuditEventListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

type Props = { events: AuditEventListResponse; jobId: string };
type Event = AuditEventListResponse["items"][number];

const Audit = ({ events, jobId }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(events.pagination);

    const key = `audit-events${table.query}&jobId=${encodeURIComponent(jobId)}`;
    const { data, isLoading } = useSWR<AuditEventListResponse>(key, {
        fallbackData: table.isInitialQuery ? events : undefined,
    });

    const columns: TableColumn<Event>[] = [
        {
            id: "created",
            label: "Created",
            render: (event) => dt(event.createdAt),
            cellClassName: "whitespace-nowrap",
        },
        { id: "action", label: "Action", isRowHeader: true, render: (event) => event.action },
        {
            id: "actor",
            label: "Actor ID",
            render: (event) => event.userId || "System",
            cellClassName: "font-mono text-xs",
        },
        {
            id: "resource",
            label: "Resource",
            render: (event) => (
                <div>
                    <div>{event.resourceType}</div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">
                        {event.resourceId || "—"}
                    </div>
                </div>
            ),
        },
        {
            id: "result",
            label: "Result",
            render: (event) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(event.result === "success" ? "succeeded" : "failed"),
                    )}
                >
                    {event.result}
                </span>
            ),
        },
        {
            id: "job",
            label: "Job",
            render: (event) =>
                event.jobId ? (
                    <Link
                        className="font-mono text-xs text-accent hover:underline"
                        href={`/jobs/${encodeURIComponent(event.jobId)}`}
                    >
                        {event.jobId}
                    </Link>
                ) : (
                    "—"
                ),
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="space-y-1 px-6 py-5">
                <h2 className="text-base font-semibold">Audit events</h2>
                <div className="text-sm text-foreground-500">
                    Requests and final outcomes. Command contents and secret metadata are not
                    displayed.
                </div>
                {jobId && (
                    <Link
                        className="inline-block text-sm text-accent hover:underline"
                        href="/audit"
                    >
                        Show all events
                    </Link>
                )}
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Audit;
