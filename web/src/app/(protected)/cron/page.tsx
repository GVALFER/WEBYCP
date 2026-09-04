import type { AccountListResponse, CronJobListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Cron from "./cron";

const CronPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/cron", searchParams);
    const [accounts, jobs] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("cron-jobs", { searchParams: query }).json<CronJobListResponse>(),
    ]);

    await syncPage("/cron", searchParams, query, jobs.pagination);

    return <Cron accounts={accounts} jobs={jobs} />;
};

export default CronPage;
