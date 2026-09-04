import type {
    AccountListResponse,
    BackupArtifactListResponse,
    BackupPlanListResponse,
    BackupRunListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPages, type PageProps } from "@/components/table/paginationServer";
import Backups from "./backups";

const BackupsPage = async ({ searchParams }: PageProps) => {
    const [plansQuery, runsQuery, artifactsQuery] = await Promise.all([
        getPageQuery("/backups", searchParams, "plans"),
        getPageQuery("/backups", searchParams, "runs"),
        getPageQuery("/backups", searchParams, "artifacts"),
    ]);

    const [accounts, plans, runs, artifacts] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("backup-plans", { searchParams: plansQuery }).json<BackupPlanListResponse>(),
        api.get("backup-runs", { searchParams: runsQuery }).json<BackupRunListResponse>(),
        api
            .get("backup-artifacts", { searchParams: artifactsQuery })
            .json<BackupArtifactListResponse>(),
    ]);

    await syncPages("/backups", searchParams, [
        { name: "plans", query: plansQuery, pagination: plans.pagination },
        { name: "runs", query: runsQuery, pagination: runs.pagination },
        { name: "artifacts", query: artifactsQuery, pagination: artifacts.pagination },
    ]);

    return <Backups accounts={accounts} plans={plans} runs={runs} artifacts={artifacts} />;
};

export default BackupsPage;
