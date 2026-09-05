"use client";

import { Server, Waypoints } from "lucide-react";
import useSWR from "swr";
import type { DNSProviderListResponse, NodeListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";

type Props = {
    nodes: NodeListResponse;
    providers: DNSProviderListResponse;
};

const Providers = ({ nodes, providers }: Props) => {
    const { dt } = useTimezone();

    const { data = providers } = useSWR<DNSProviderListResponse>("dns/providers", {
        fallbackData: providers,
    });

    const nodeNames = new Map(nodes.items.map((item) => [item.id, item.name]));

    return (
        <section className="panel-card overflow-hidden">
            <div className="border-b border-divider px-6 py-5">
                <h2 className="text-base font-semibold">Authoritative DNS providers</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Driver registrations and live health reported by each Agent.
                </div>
            </div>
            <div className="divide-y divide-divider">
                {data.items.map((provider) => (
                    <div
                        key={provider.id}
                        className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                    >
                        <div className="flex items-center gap-4">
                            <div className="icon-box">
                                <Waypoints className="size-5" aria-hidden="true" />
                            </div>
                            <div>
                                <div className="flex items-center gap-2 font-medium">
                                    {provider.name}
                                    <span
                                        className={cn(
                                            "rounded-full px-2 py-0.5 text-xs capitalize",
                                            statusClass(provider.status),
                                        )}
                                    >
                                        {provider.status}
                                    </span>
                                </div>
                                <div className="mt-1 text-xs text-foreground-400">
                                    {provider.driver} · registered {dt(provider.createdAt)}
                                </div>
                            </div>
                        </div>
                        <div className="flex items-center gap-2 text-sm text-foreground-500">
                            <Server className="size-4" aria-hidden="true" />
                            {nodeNames.get(provider.nodeId) ?? provider.nodeId}
                        </div>
                    </div>
                ))}
                {!data.items.length && (
                    <div className="px-6 py-12 text-center text-sm text-foreground-400">
                        No DNS providers registered.
                    </div>
                )}
            </div>
        </section>
    );
};

export default Providers;
