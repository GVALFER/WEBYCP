"use client";

import { HardDrive } from "lucide-react";
import useSWR from "swr";
import CheckNode from "@/components/actions/checkNode";
import type { NodeListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";

type Props = { nodes: NodeListResponse };

const Destinations = ({ nodes }: Props) => {
    const { dt } = useTimezone();
    const { data = nodes } = useSWR<NodeListResponse>("nodes", { fallbackData: nodes });

    return (
        <div className="space-y-6">
            <div className="rounded-xl border border-warning/25 bg-warning/8 px-5 py-4 text-sm text-foreground-500">
                Local backups are stored on the account&apos;s server. They do not protect against loss
                of that server. Remote destinations are not available yet.
            </div>
            {data.items.map((node) => (
                <section key={node.id} className="panel-card overflow-hidden">
                    <div className="flex flex-wrap items-center justify-between gap-4 px-6 py-5">
                        <div className="flex items-center gap-4">
                            <div className="icon-box"><HardDrive className="size-5" /></div>
                            <div>
                                <h2 className="text-base font-semibold">{node.name}</h2>
                                <div className="mt-1 text-xs text-foreground-400">
                                    Last checked: {node.capabilitiesAt ? dt(node.capabilitiesAt) : "Never"} · Node {node.status}
                                </div>
                            </div>
                        </div>
                        <CheckNode node={node} />
                    </div>
                    <div className="space-y-3 px-6 pb-6">
                        {node.capabilities ? node.capabilities.backups.map((storage) => (
                            <div key={storage.driver} className="rounded-xl border border-divider bg-surface-secondary/45 p-4">
                                <div className="flex items-center gap-3">
                                    <div className="font-medium capitalize">{storage.driver}</div>
                                    <span className={cn("text-xs capitalize", storage.status === "healthy" ? "text-success" : "text-danger")}>
                                        {storage.status}
                                    </span>
                                </div>
                                {storage.driver === "local" && (
                                    <div className="mt-2 space-y-2 text-sm text-foreground-500">
                                        <div className="break-all font-mono text-xs">/var/backups/webycp/&lt;account-id&gt;/&lt;run-id&gt;.tar.gz</div>
                                        <div>Root-only archives with SHA-256 verification and per-plan retention.</div>
                                    </div>
                                )}
                            </div>
                        )) : (
                            <div className="text-sm text-foreground-400">Backup storage has not been checked yet.</div>
                        )}
                    </div>
                </section>
            ))}
        </div>
    );
};

export default Destinations;
