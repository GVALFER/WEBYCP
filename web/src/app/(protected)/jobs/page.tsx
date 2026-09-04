import type { JobListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Jobs from "./jobs";

const JobsPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/jobs", searchParams);
    const jobs = await api.get("jobs", { searchParams: query }).json<JobListResponse>();

    await syncPage("/jobs", searchParams, query, jobs.pagination);

    return <Jobs jobs={jobs} />;
};

export default JobsPage;
