"use client";

import { HardDrive } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    BackupArtifactListResponse,
    BackupPlanListResponse,
    BackupRunListResponse,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import BackupArtifactActions from "./actions/backupArtifactActions";
import BackupPlanActions from "./actions/backupPlanActions";
import CreateBackup from "./actions/createBackup";

type BackupsProps = {
    accounts: AccountListResponse;
    plans: BackupPlanListResponse;
    runs: BackupRunListResponse;
    artifacts: BackupArtifactListResponse;
};

type Plan = BackupPlanListResponse["items"][number];
type Run = BackupRunListResponse["items"][number];
type Artifact = BackupArtifactListResponse["items"][number];

const Backups = ({ accounts, plans, runs, artifacts }: BackupsProps) => {
    const { dt } = useTimezone();

    const plansTable = useTable(plans.pagination, "plans");
    const runsTable = useTable(runs.pagination, "runs");
    const artifactsTable = useTable(artifacts.pagination, "artifacts");

    const { data: planData } = useSWR<BackupPlanListResponse>(`backup-plans${plansTable.query}`, {
        fallbackData: plansTable.isInitialQuery ? plans : undefined,
    });
    const { data: runData } = useSWR<BackupRunListResponse>(`backup-runs${runsTable.query}`, {
        fallbackData: runsTable.isInitialQuery ? runs : undefined,
    });
    const { data: artifactData } = useSWR<BackupArtifactListResponse>(
        `backup-artifacts${artifactsTable.query}`,
        { fallbackData: artifactsTable.isInitialQuery ? artifacts : undefined },
    );

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
                            {plan.includeFiles && plan.includeDatabases
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
            render: (plan) => plan.schedule || "Manual only",
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
            render: (plan) => <BackupPlanActions plan={plan} />,
        },
    ];

    const artifactColumns: TableColumn<Artifact>[] = [
        {
            id: "artifact",
            label: "Artifact",
            isRowHeader: true,
            render: (artifact) => (
                <div>
                    <div className="font-medium">{dt(artifact.createdAt)}</div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">{artifact.id}</div>
                </div>
            ),
        },
        {
            id: "size",
            label: "Size",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (artifact) => `${(artifact.size / 1_048_576).toFixed(2)} MB`,
        },
        {
            id: "checksum",
            label: "SHA-256",
            cellClassName: "font-mono text-xs text-foreground-500",
            render: (artifact) => `${artifact.checksum.slice(0, 12)}…`,
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (artifact) => <BackupArtifactActions artifact={artifact} />,
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
                    <CreateBackup accounts={accounts} />
                </div>
                <Table table={plansTable} columns={planColumns} data={planData} />
            </section>

            <section className="panel-card overflow-hidden">
                <div className="px-6 py-5">
                    <h2 className="text-base font-semibold">Verified artifacts</h2>
                </div>
                <Table table={artifactsTable} columns={artifactColumns} data={artifactData} />
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

export default Backups;
