"use client";

import { Clock3, Server } from "lucide-react";
import useSWR from "swr";
import type { DNSSettings } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import EditNameservers from "./actions/editNameservers";

type ValueProps = {
    icon: typeof Server;
    label: string;
    value: string;
};

const Nameservers = ({ settings }: { settings: DNSSettings }) => {
    const { dt } = useTimezone();

    const { data = settings } = useSWR<DNSSettings>("dns/settings", {
        fallbackData: settings,
    });

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Authoritative defaults</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Used when WEBYCP creates a new DNS zone.
                    </div>
                </div>
                <EditNameservers settings={data} />
            </div>
            <div className="grid gap-4 px-6 py-5 sm:grid-cols-2 xl:grid-cols-4">
                <Value
                    icon={Server}
                    label="Primary nameserver"
                    value={data.primaryNameserver || "Not configured"}
                />
                <Value
                    icon={Server}
                    label="Secondary nameserver"
                    value={data.secondaryNameserver || "Not configured"}
                />
                <Value icon={Clock3} label="Default TTL" value={`${data.defaultTtl} seconds`} />
                <Value icon={Clock3} label="Updated" value={dt(data.updatedAt)} />
            </div>
        </section>
    );
};

const Value = ({ icon: Icon, label, value }: ValueProps) => (
    <div className="rounded-xl border border-divider bg-surface-secondary/45 p-4">
        <div className="flex items-center gap-2 text-xs text-foreground-400">
            <Icon className="size-4" aria-hidden="true" />
            {label}
        </div>
        <div className="mt-3 break-all font-medium">{value}</div>
    </div>
);

export default Nameservers;
