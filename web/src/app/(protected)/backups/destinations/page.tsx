import type { NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Destinations from "./destinations";

const DestinationsPage = async () => {
    const nodes = await api.get("nodes").json<NodeListResponse>();

    return <Destinations nodes={nodes} />;
};

export default DestinationsPage;
