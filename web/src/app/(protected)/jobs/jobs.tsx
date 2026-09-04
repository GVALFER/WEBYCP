"use client";

import { Clock3 } from "lucide-react";
import useSWR from "swr";
import type { JobListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

type JobsProps = {
    jobs: JobListResponse;
};

const Jobs = ({ jobs }: JobsProps) => {
    const { dt } = useTimezone();

    const { data } = useSWR<JobListResponse>("jobs", {
        fallbackData: jobs,
    });

    return (
        <section className="panel-card overflow-hidden">
            <div className="border-b border-divider px-6 py-5">
                <h2 className="text-base font-semibold">Recent jobs</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Durable operations executed by the local agent.
                </div>
            </div>
            <div className="divide-y divide-divider">
                {data?.items.length ? (
                    data.items.map((job) => (
                        <div
                            key={job.id}
                            className="grid gap-3 px-6 py-5 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-center sm:gap-8"
                        >
                            <div className="flex min-w-0 items-center gap-3">
                                <div className="icon-box size-9">
                                    <Clock3 className="size-4" />
                                </div>
                                <div className="min-w-0">
                                    <div className="truncate font-medium">{job.kind}</div>
                                    <div className="truncate font-mono text-xs text-foreground-400">
                                        {job.id}
                                    </div>
                                </div>
                            </div>
                            <div className="text-sm text-foreground-500">{dt(job.createdAt)}</div>
                            <span
                                className={cn(
                                    "w-fit rounded-full px-2.5 py-1 text-xs capitalize",
                                    statusClass(job.status),
                                )}
                            >
                                {job.status}
                            </span>
                        </div>
                    ))
                ) : (
                    <div className="px-6 py-12 text-center text-sm text-foreground-400">
                        No jobs have run yet.
                    </div>
                )}
            </div>
        </section>
    );
};

export default Jobs;
