import { JobListResponse, NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Dashboard from "./dashboard";

const OverviewPage = async () => {
    const [nodes, jobs] = await Promise.all([
        api.get("nodes").json<NodeListResponse>(),
        api.get("jobs").json<JobListResponse>(),
    ]);

    return <Dashboard nodes={nodes} jobs={jobs} />;
};

export default OverviewPage;
