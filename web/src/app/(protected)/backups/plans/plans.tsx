"use client";

import { HardDrive } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    BackupPlanListResponse,
    BackupRunListResponse,
    ServiceSettings,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import BackupPlanActions from "./actions/backupPlanActions";
import PlanForm from "./actions/planForm";

type Props = {
    accounts: AccountListResponse;
    plans: BackupPlanListResponse;
    runs: BackupRunListResponse;
    settings: ServiceSettings;
};

type Plan = BackupPlanListResponse["items"][number];
type Run = BackupRunListResponse["items"][number];

const Plans = ({ accounts, plans, runs, settings: initialSettings }: Props) => {
    const { dt } = useTimezone();

    const plansTable = useTable(plans.pagination, "plans");
    const runsTable = useTable(runs.pagination, "runs");

    const { data: planData } = useSWR<BackupPlanListResponse>(`backup-plans${plansTable.query}`, {
        fallbackData: plansTable.isInitialQuery ? plans : undefined,
    });
    const { data: runData } = useSWR<BackupRunListResponse>(`backup-runs${runsTable.query}`, {
        fallbackData: runsTable.isInitialQuery ? runs : undefined,
    });
    const { data: settings = initialSettings } = useSWR<ServiceSettings>("service-settings", {
        fallbackData: initialSettings,
    });

    const planColumns: TableColumn<Plan>[] = [
        {
            id: "plan",
            label: "Plan",
            isRowHeader: true,
            render: (plan) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <HardDrive className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{plan.name}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {plan.storageDriver} · {plan.includeFiles && plan.includeDatabases
                                ? "Files and databases"
                                : plan.includeFiles
                                  ? "Files"
                                  : "Databases"}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "schedule",
            label: "Schedule",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (plan) => !plan.enabled ? "Disabled" : plan.schedule || "Manual only",
        },
        {
            id: "nextRun",
            label: "Next run",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (plan) => (plan.nextRunAt ? dt(plan.nextRunAt) : "—"),
        },
        {
            id: "retention",
            label: "Retention",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (plan) => `${plan.retentionCount} backups`,
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (plan) => (
                <div className="flex items-center gap-2">
                    <PlanForm accounts={accounts} driver={plan.storageDriver} plan={plan} />
                    <BackupPlanActions plan={plan} />
                </div>
            ),
        },
    ];

    const runColumns: TableColumn<Run>[] = [
        {
            id: "run",
            label: "Run",
            isRowHeader: true,
            render: (run) => (
                <div>
                    <div className="font-medium">{dt(run.createdAt)}</div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">{run.id}</div>
                </div>
            ),
        },
        {
            id: "status",
            label: "Status",
            render: (run) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(run.status),
                    )}
                >
                    {run.status}
                </span>
            ),
        },
        {
            id: "finished",
            label: "Finished",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (run) => (run.finishedAt ? dt(run.finishedAt) : "—"),
        },
        {
            id: "message",
            label: "Message",
            cellClassName: "max-w-sm truncate text-foreground-500",
            render: (run) => run.error || "—",
        },
    ];

    return (
        <div className="space-y-6">
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Backup plans</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Scheduled and on-demand local backups with retention.
                        </div>
                    </div>
                    <PlanForm accounts={accounts} driver={settings.defaults.backupDriver} />
                </div>
                <Table table={plansTable} columns={planColumns} data={planData} />
            </section>

            <section className="panel-card overflow-hidden">
                <div className="px-6 py-5">
                    <h2 className="text-base font-semibold">Recent runs</h2>
                </div>
                <Table table={runsTable} columns={runColumns} data={runData} />
            </section>
        </div>
    );
};

export default Plans;
