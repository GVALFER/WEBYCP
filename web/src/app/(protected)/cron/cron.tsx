"use client";

import { Clock3 } from "lucide-react";
import useSWR from "swr";
import type { AccountListResponse, CronJobListResponse } from "@/contracts/types";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreateCron from "./actions/createCron";
import CronActions from "./actions/cronActions";

type CronProps = {
    accounts: AccountListResponse;
    jobs: CronJobListResponse;
};

const Cron = ({ accounts, jobs }: CronProps) => {
    const { data } = useSWR<CronJobListResponse>("cron-jobs", {
        fallbackData: jobs,
    });

    return (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <section className="panel-card overflow-hidden">
                <div className="border-b border-divider px-6 py-5">
                    <h2 className="text-base font-semibold">Cron jobs</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Commands run as the hosting account from its home directory.
                    </div>
                </div>
                <div className="divide-y divide-divider">
                    {data?.items.length ? (
                        data.items.map((item) => (
                                <div
                                    key={item.id}
                                    className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                                >
                                    <div className="flex min-w-0 items-center gap-4">
                                        <div className="icon-box">
                                            <Clock3 className="size-5" />
                                        </div>
                                        <div className="min-w-0">
                                            <div className="font-medium">{item.name}</div>
                                            <div className="mt-1 truncate font-mono text-xs text-foreground-400">
                                                {item.schedule} · {item.command}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span
                                            className={cn(
                                                "rounded-full px-2 py-1 text-xs capitalize",
                                                statusClass(item.status),
                                            )}
                                        >
                                            {item.status}
                                        </span>
                                        <CronActions item={item} />
                                    </div>
                                </div>
                            ))
                    ) : (
                        <div className="px-6 py-12 text-center text-sm text-foreground-400">
                            No cron jobs yet.
                        </div>
                    )}
                </div>
            </section>

            <CreateCron accounts={accounts} />
        </div>
    );
};

export default Cron;
