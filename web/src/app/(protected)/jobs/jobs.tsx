"use client";

import { Clock3 } from "lucide-react";
import Link from "next/link";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { JobListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

type JobsProps = {
    jobs: JobListResponse;
};

type Job = JobListResponse["items"][number];

const Jobs = ({ jobs }: JobsProps) => {
    const { dt } = useTimezone();
    const table = useTable(jobs.pagination);

    const { data, isLoading } = useSWR<JobListResponse>(`jobs${table.query}`, {
        fallbackData: table.isInitialQuery ? jobs : undefined,
    });

    const columns: TableColumn<Job>[] = [
        {
            id: "operation",
            label: "Operation",
            isRowHeader: true,
            render: (job) => (
                <div className="flex min-w-0 items-center gap-3">
                    <div className="icon-box size-9">
                        <Clock3 className="size-4" aria-hidden="true" />
                    </div>
                    <div className="min-w-0">
                        <Link
                            className="block truncate font-medium text-accent hover:underline"
                            href={`/jobs/${encodeURIComponent(job.id)}`}
                        >
                            {job.kind}
                        </Link>
                        <div className="truncate font-mono text-xs text-foreground-400">
                            {job.id}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (job) => dt(job.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (job) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(job.status),
                    )}
                >
                    {job.status}
                </span>
            ),
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="px-6 py-5">
                <h2 className="text-base font-semibold">Recent jobs</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Durable operations executed by the local agent.
                </div>
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Jobs;
