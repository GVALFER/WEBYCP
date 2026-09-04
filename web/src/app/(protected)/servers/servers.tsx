"use client";

import { Server } from "lucide-react";
import useSWR from "swr";
import CheckNode from "@/components/actions/checkNode";
import CapabilityList from "@/components/services/capabilityList";
import type { NodeListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

type Props = {
    nodes: NodeListResponse;
};

const Servers = ({ nodes: initial }: Props) => {
    const { dt } = useTimezone();

    const { data = initial } = useSWR<NodeListResponse>("nodes", {
        fallbackData: initial,
    });

    return (
        <div className="space-y-6">
            {data.items.map((node) => (
                <section key={node.id} className="panel-card overflow-hidden">
                    <div className="flex flex-col gap-4 border-b border-divider px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
                        <div className="flex min-w-0 items-center gap-4">
                            <div className="icon-box">
                                <Server className="size-5" aria-hidden="true" />
                            </div>
                            <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                    <h2 className="truncate text-base font-semibold">
                                        {node.name}
                                    </h2>
                                    <span
                                        className={cn(
                                            "rounded-full px-2.5 py-1 text-xs capitalize",
                                            statusClass(node.status),
                                        )}
                                    >
                                        {node.status}
                                    </span>
                                </div>
                                <div className="mt-1 truncate text-sm text-foreground-500">
                                    {node.kind} · {node.endpoint}
                                </div>
                            </div>
                        </div>
                        <CheckNode node={node} />
                    </div>

                    <div className="space-y-5 px-6 py-5">
                        <div className="flex flex-wrap gap-x-8 gap-y-2 text-xs text-foreground-400">
                            <div>Last seen: {node.lastSeenAt ? dt(node.lastSeenAt) : "Never"}</div>
                            <div>
                                Services checked:{" "}
                                {node.capabilitiesAt ? dt(node.capabilitiesAt) : "Never"}
                            </div>
                        </div>
                        {node.capabilities ? (
                            <CapabilityList capabilities={node.capabilities} />
                        ) : (
                            <div className="rounded-xl border border-warning/25 bg-warning/8 px-4 py-3 text-sm text-warning">
                                Service capabilities have not been checked yet.
                            </div>
                        )}
                    </div>
                </section>
            ))}
        </div>
    );
};

export default Servers;
