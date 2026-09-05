"use client";

import Link from "next/link";
import useSWR from "swr";
import type { JobDetail } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

const Job = ({ detail }: { detail: JobDetail }) => {
    const { dt } = useTimezone();

    const { data = detail } = useSWR<JobDetail>(`jobs/${encodeURIComponent(detail.job.id)}`, {
        fallbackData: detail,
    });

    const { job, steps } = data;

    return (
        <div className="space-y-6">
            <section className="panel-card space-y-5 p-6">
                <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h2 className="text-base font-semibold">{job.kind}</h2>
                        <div className="mt-1 break-all font-mono text-xs text-foreground-400">
                            {job.id}
                        </div>
                    </div>
                    <span
                        className={cn(
                            "rounded-full px-2.5 py-1 text-xs capitalize",
                            statusClass(job.status),
                        )}
                    >
                        {job.status}
                    </span>
                </div>
                <div className="grid gap-4 text-sm sm:grid-cols-2 xl:grid-cols-4">
                    <div>
                        Created<div className="mt-1 text-foreground-500">{dt(job.createdAt)}</div>
                    </div>
                    <div>
                        Started
                        <div className="mt-1 text-foreground-500">
                            {job.startedAt ? dt(job.startedAt) : "Not started"}
                        </div>
                    </div>
                    <div>
                        Finished
                        <div className="mt-1 text-foreground-500">
                            {job.finishedAt ? dt(job.finishedAt) : "Not finished"}
                        </div>
                    </div>
                    <div>
                        Attempts
                        <div className="mt-1 text-foreground-500">
                            {job.attempts} / {job.maxAttempts}
                        </div>
                    </div>
                </div>
                {job.error && (
                    <div className="wrap-break-word rounded-xl bg-danger/10 p-4 text-sm text-danger">
                        {job.error}
                    </div>
                )}
                <Link
                    className="inline-block text-sm text-accent hover:underline"
                    href={`/audit?jobId=${encodeURIComponent(job.id)}`}
                >
                    View request and outcome in Audit Log
                </Link>
            </section>
            <section className="panel-card overflow-hidden">
                <h2 className="px-6 py-5 text-base font-semibold">Execution steps</h2>
                <div className="divide-y divide-divider">
                    {steps.map((step) => (
                        <div key={step.id} className="space-y-2 px-6 py-4">
                            <div className="flex items-center justify-between gap-3">
                                <div className="text-sm font-medium">{step.name}</div>
                                <span
                                    className={cn(
                                        "rounded-full px-2 py-1 text-xs capitalize",
                                        statusClass(step.status),
                                    )}
                                >
                                    {step.status}
                                </span>
                            </div>
                            <div className="text-xs text-foreground-500">
                                {dt(step.startedAt)}
                                {step.finishedAt ? ` — ${dt(step.finishedAt)}` : ""}
                            </div>
                            {step.message && (
                                <div className="wrap-break-word text-sm text-foreground-500">
                                    {step.message}
                                </div>
                            )}
                        </div>
                    ))}
                    {!steps.length && (
                        <div className="px-6 py-8 text-sm text-foreground-500">
                            Execution has not started.
                        </div>
                    )}
                </div>
            </section>
        </div>
    );
};

export default Job;
