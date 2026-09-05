"use client";

import { PackageOpen } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { PackageListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import CreatePackage from "./actions/createPackage";
import PackageActions from "./actions/packageActions";

type Props = {
    packages: PackageListResponse;
};

type Package = PackageListResponse["items"][number];

const Packages = ({ packages }: Props) => {
    const { dt } = useTimezone();
    const table = useTable(packages.pagination);

    const { data, isLoading } = useSWR<PackageListResponse>(`packages${table.query}`, {
        fallbackData: table.isInitialQuery ? packages : undefined,
    });

    const columns: TableColumn<Package>[] = [
        {
            id: "package",
            label: "Package",
            isRowHeader: true,
            render: (item) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <PackageOpen className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{item.name}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {item.accountCount} {item.accountCount === 1 ? "Account" : "Accounts"}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "web",
            label: "Web",
            cellClassName: "min-w-44",
            render: (item) => (
                <Limits
                    values={[
                        ["Websites", item.limits.websites],
                        ["Domains", item.limits.domains],
                        ["Aliases", item.limits.aliases],
                    ]}
                />
            ),
        },
        {
            id: "data",
            label: "Data",
            cellClassName: "min-w-44",
            render: (item) => (
                <Limits
                    values={[
                        ["Databases", item.limits.databases],
                        ["Database users", item.limits.databaseUsers],
                    ]}
                />
            ),
        },
        {
            id: "operations",
            label: "Operations",
            cellClassName: "min-w-44",
            render: (item) => (
                <Limits
                    values={[
                        ["Scheduled tasks", item.limits.scheduledTasks],
                        ["Backup plans", item.limits.backupPlans],
                        ["Backup retention", item.limits.backupRetention],
                    ]}
                />
            ),
        },
        {
            id: "updated",
            label: "Updated",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (item) => dt(item.updatedAt),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (item) => <PackageActions value={item} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">Hosting Packages</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Account resource limits enforced by the control plane.
                    </div>
                </div>
                <CreatePackage />
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

type LimitsProps = {
    values: [string, number][];
};

const Limits = ({ values }: LimitsProps) => (
    <div className="space-y-1 text-xs text-foreground-500">
        {values.map(([label, limit]) => (
            <div key={label} className="flex justify-between gap-4">
                <span>{label}</span>
                <span className="font-medium tabular-nums text-foreground">{limit}</span>
            </div>
        ))}
    </div>
);

export default Packages;
