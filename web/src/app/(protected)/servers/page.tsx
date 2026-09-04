import type { NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Servers from "./servers";

const ServersPage = async () => {
    const nodes = await api.get("nodes").json<NodeListResponse>();

    return <Servers nodes={nodes} />;
};

export default ServersPage;
