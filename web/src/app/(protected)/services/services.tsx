"use client";

import useSWR from "swr";
import CapabilityList from "@/components/services/capabilityList";
import type { NodeListResponse, ServiceSettings } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import EditDefaults from "./actions/editDefaults";

type Props = {
    nodes: NodeListResponse;
    settings: ServiceSettings;
};

const Services = ({ nodes: initialNodes, settings: initialSettings }: Props) => {
    const { dt } = useTimezone();

    const { data: nodes = initialNodes } = useSWR<NodeListResponse>("nodes", {
        fallbackData: initialNodes,
    });
    const { data: settings = initialSettings } = useSWR<ServiceSettings>("service-settings", {
        fallbackData: initialSettings,
    });

    return (
        <div className="space-y-6">
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Service defaults</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Initial selections for new resources. Existing resources keep their own
                            drivers.
                        </div>
                    </div>
                    <EditDefaults settings={settings} />
                </div>
                <div className="grid gap-4 px-6 py-5 sm:grid-cols-2 xl:grid-cols-3">
                    <Default label="Web server" value={settings.defaults.webDriver} />
                    <Default
                        label="Runtime"
                        value={`${settings.defaults.runtimeDriver} ${settings.defaults.runtimeVersion}`}
                    />
                    <Default label="Database" value={settings.defaults.databaseDriver} />
                    <Default label="Scheduler" value={settings.defaults.schedulerDriver} />
                    <Default label="Backup storage" value={settings.defaults.backupDriver} />
                    <Default label="Updated" value={dt(settings.updatedAt)} />
                </div>
            </section>

            {nodes.items.map((node) => (
                <section key={node.id} className="panel-card overflow-hidden">
                    <div className="border-b border-divider px-6 py-5">
                        <h2 className="text-base font-semibold">{node.name}</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Observed services ·{" "}
                            {node.capabilitiesAt ? dt(node.capabilitiesAt) : "not checked"}
                        </div>
                    </div>
                    <div className="px-6 py-5">
                        {node.capabilities ? (
                            <CapabilityList capabilities={node.capabilities} />
                        ) : (
                            <div className="text-sm text-foreground-500">
                                No observed service capabilities.
                            </div>
                        )}
                    </div>
                </section>
            ))}
        </div>
    );
};

const Default = ({ label, value }: { label: string; value: string }) => (
    <div className="rounded-xl border border-divider bg-surface-secondary/45 p-4">
        <div className="text-xs text-foreground-400">{label}</div>
        <div className="mt-2 font-medium">{value}</div>
    </div>
);

export default Services;
