import type { NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Dashboard from "./dashboard";

const OverviewPage = async () => {
    const nodes = await api.get("nodes").json<NodeListResponse>();
    return <Dashboard nodes={nodes} />;
};

export default OverviewPage;
