"use client";

import { Clock3 } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { AccountListResponse, CronJobListResponse } from "@/contracts/types";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreateCron from "./actions/createCron";
import CronActions from "./actions/cronActions";

type CronProps = {
    accounts: AccountListResponse;
    jobs: CronJobListResponse;
};

type CronJob = CronJobListResponse["items"][number];

const Cron = ({ accounts, jobs }: CronProps) => {
    const table = useTable(jobs.pagination);

    const { data } = useSWR<CronJobListResponse>(`cron-jobs${table.query}`, {
        fallbackData: table.isInitialQuery ? jobs : undefined,
    });

    const columns: TableColumn<CronJob>[] = [
        {
            id: "job",
            label: "Task",
            isRowHeader: true,
            render: (job) => (
                <div className="flex min-w-0 items-center gap-4">
                    <div className="icon-box">
                        <Clock3 className="size-5" aria-hidden="true" />
                    </div>
                    <div className="min-w-0">
                        <div className="font-medium">{job.name}</div>
                        <div className="mt-1 max-w-xl truncate font-mono text-xs text-foreground-400">
                            {job.command}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "schedule",
            label: "Schedule",
            cellClassName: "whitespace-nowrap font-mono text-xs text-foreground-500",
            render: (job) => job.schedule,
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
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (job) => <CronActions item={job} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Scheduled tasks</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Commands run as the hosting account from its home directory.
                    </div>
                </div>
                <CreateCron accounts={accounts} />
            </div>
            <Table table={table} columns={columns} data={data} />
        </section>
    );
};

export default Cron;
