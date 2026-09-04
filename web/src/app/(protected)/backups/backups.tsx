"use client";

import { HardDrive } from "lucide-react";
import useSWR from "swr";
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

const Backups = ({
    accounts,
    plans: initialPlans,
    runs: initialRuns,
    artifacts: initialArtifacts,
}: BackupsProps) => {
    const { dt } = useTimezone();

    const { data: plans } = useSWR<BackupPlanListResponse>("backup-plans", {
        fallbackData: initialPlans,
    });
    const { data: runs } = useSWR<BackupRunListResponse>("backup-runs", {
        fallbackData: initialRuns,
    });
    const { data: artifacts } = useSWR<BackupArtifactListResponse>("backup-artifacts", {
        fallbackData: initialArtifacts,
    });

    return (
        <div className="space-y-6">
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Backup plans</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Scheduled and on-demand local backups with retention.
                        </div>
                    </div>
                    <CreateBackup accounts={accounts} />
                </div>
                <div className="divide-y divide-divider">
                    {plans?.items.length ? (
                        plans.items.map((plan) => (
                            <div
                                key={plan.id}
                                className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                            >
                                <div className="flex items-center gap-4">
                                    <div className="icon-box">
                                        <HardDrive className="size-5" />
                                    </div>
                                    <div>
                                        <div className="font-medium">{plan.name}</div>
                                        <div className="mt-1 text-xs text-foreground-400">
                                            {plan.schedule || "Manual only"} · keep{" "}
                                            {plan.retentionCount}
                                            {plan.nextRunAt
                                                ? ` · next ${dt(plan.nextRunAt)}`
                                                : ""}
                                        </div>
                                    </div>
                                </div>
                                <BackupPlanActions plan={plan} />
                            </div>
                        ))
                    ) : (
                        <div className="px-6 py-12 text-center text-sm text-foreground-400">
                            No backup plans yet.
                        </div>
                    )}
                </div>
            </section>

            <section className="panel-card overflow-hidden">
                <div className="border-b border-divider px-6 py-5">
                    <h2 className="text-base font-semibold">Verified artifacts</h2>
                </div>
                <div className="divide-y divide-divider">
                    {artifacts?.items.length ? (
                        artifacts.items.map((artifact) => (
                            <div
                                key={artifact.id}
                                className="flex flex-col gap-4 px-6 py-4 sm:flex-row sm:items-center sm:justify-between"
                            >
                                <div>
                                    <div className="font-medium">{dt(artifact.createdAt)}</div>
                                    <div className="mt-1 text-xs text-foreground-400">
                                        {(artifact.size / 1_048_576).toFixed(2)} MB · SHA-256{" "}
                                        {artifact.checksum.slice(0, 12)}…
                                    </div>
                                </div>
                                <BackupArtifactActions artifact={artifact} />
                            </div>
                        ))
                    ) : (
                        <div className="px-6 py-10 text-center text-sm text-foreground-400">
                            No completed artifacts yet.
                        </div>
                    )}
                </div>
            </section>

            <section className="panel-card p-6">
                <h2 className="text-base font-semibold">Recent runs</h2>
                <div className="mt-4 space-y-2">
                    {runs?.items.length ? (
                        runs.items.slice(0, 10).map((item) => (
                            <div
                                key={item.id}
                                className="flex items-center justify-between rounded-xl border border-border/70 bg-surface-secondary/60 px-4 py-3 text-sm"
                            >
                                <div>{dt(item.createdAt)}</div>
                                <span
                                    className={cn(
                                        "rounded-full px-2 py-1 text-xs capitalize",
                                        statusClass(item.status),
                                    )}
                                >
                                    {item.status}
                                </span>
                            </div>
                        ))
                    ) : (
                        <div className="py-6 text-center text-sm text-foreground-400">
                            No backup runs yet.
                        </div>
                    )}
                </div>
            </section>
        </div>
    );
};

export default Backups;
