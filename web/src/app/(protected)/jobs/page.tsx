import type { JobListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Jobs from "./jobs";

const JobsPage = async () => {
    const jobs = await api.get("jobs").json<JobListResponse>();

    return <Jobs jobs={jobs} />;
};

export default JobsPage;
