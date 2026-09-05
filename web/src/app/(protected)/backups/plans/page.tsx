import type {
    AccountListResponse,
    BackupPlanListResponse,
    BackupRunListResponse,
    ServiceSettings,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPages, type PageProps } from "@/components/table/paginationServer";
import Plans from "./plans";

const PlansPage = async ({ searchParams }: PageProps) => {
    const [plansQuery, runsQuery] = await Promise.all([
        getPageQuery("/backups/plans", searchParams, "plans"),
        getPageQuery("/backups/plans", searchParams, "runs"),
    ]);

    const [accounts, plans, runs, settings] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("backup-plans", { searchParams: plansQuery }).json<BackupPlanListResponse>(),
        api.get("backup-runs", { searchParams: runsQuery }).json<BackupRunListResponse>(),
        api.get("service-settings").json<ServiceSettings>(),
    ]);

    await syncPages("/backups/plans", searchParams, [
        { name: "plans", query: plansQuery, pagination: plans.pagination },
        { name: "runs", query: runsQuery, pagination: runs.pagination },
    ]);

    return (
        <Plans
            accounts={accounts}
            plans={plans}
            runs={runs}
            settings={settings}
        />
    );
};

export default PlansPage;
