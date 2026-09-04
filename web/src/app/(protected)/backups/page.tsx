import type {
    AccountListResponse,
    BackupArtifactListResponse,
    BackupPlanListResponse,
    BackupRunListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import Backups from "./backups";

const BackupsPage = async () => {
    const [accounts, plans, runs, artifacts] = await Promise.all([
        api.get("accounts").json<AccountListResponse>(),
        api.get("backup-plans").json<BackupPlanListResponse>(),
        api.get("backup-runs").json<BackupRunListResponse>(),
        api.get("backup-artifacts").json<BackupArtifactListResponse>(),
    ]);

    return <Backups accounts={accounts} plans={plans} runs={runs} artifacts={artifacts} />;
};

export default BackupsPage;
