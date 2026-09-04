import type { AccountListResponse, CronJobListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Cron from "./cron";

const CronPage = async () => {
    const [accounts, jobs] = await Promise.all([
        api.get("accounts").json<AccountListResponse>(),
        api.get("cron-jobs").json<CronJobListResponse>(),
    ]);

    return <Cron accounts={accounts} jobs={jobs} />;
};

export default CronPage;
