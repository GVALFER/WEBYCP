"use client";

import Link from "next/link";
import { Activity, Server } from "lucide-react";
import useSWR from "swr";
import CheckNode from "@/components/actions/checkNode";
import CapabilityList from "@/components/services/capabilityList";
import type { NodeListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

const Dashboard = ({ nodes }: { nodes: NodeListResponse }) => {
    const { dt } = useTimezone();

    const { data = nodes } = useSWR<NodeListResponse>("nodes", { fallbackData: nodes });

    const online = data.items.filter((node) => node.status === "online").length;

    return (
        <div className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2">
                <div className="metric-card">
                    <Server className="mb-4 size-5 text-accent" />
                    <div className="text-sm text-foreground-500">Managed servers</div>
                    <div className="mt-2 text-2xl font-semibold">{data.items.length}</div>
                </div>
                <div className="metric-card">
                    <Activity className="mb-4 size-5 text-accent" />
                    <div className="text-sm text-foreground-500">Reachable at last check</div>
                    <div className="mt-2 text-2xl font-semibold">
                        {online} / {data.items.length}
                    </div>
                </div>
            </div>
            <div className="text-sm text-foreground-500">
                Latest observed state, not live monitoring. Operation history is available in{" "}
                <Link href="/jobs" className="text-accent hover:underline">
                    Jobs
                </Link>{" "}
                and{" "}
                <Link href="/audit" className="text-accent hover:underline">
                    Audit Log
                </Link>
                .
            </div>
            {data.items.map((node) => (
                <section key={node.id} className="panel-card overflow-hidden">
                    <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-5">
                        <div>
                            <div className="flex items-center gap-3">
                                <h2 className="text-base font-semibold">{node.name}</h2>
                                <span
                                    className={cn(
                                        "rounded-full px-2.5 py-1 text-xs capitalize",
                                        statusClass(node.status),
                                    )}
                                >
                                    {node.status}
                                </span>
                            </div>
                            <div className="mt-1 text-xs text-foreground-400">
                                Services checked:{" "}
                                {node.capabilitiesAt ? dt(node.capabilitiesAt) : "Never"}
                            </div>
                        </div>
                        <CheckNode node={node} />
                    </div>
                    <div className="border-t border-divider p-6">
                        {node.capabilities ? (
                            <CapabilityList capabilities={node.capabilities} />
                        ) : (
                            <div className="text-sm text-foreground-500">
                                Service capabilities have not been checked yet.
                            </div>
                        )}
                    </div>
                </section>
            ))}
        </div>
    );
};

export default Dashboard;
