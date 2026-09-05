import type { DNSProviderListResponse, NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Providers from "./providers";

const ProvidersPage = async () => {
    const [providers, nodes] = await Promise.all([
        api.get("dns/providers").json<DNSProviderListResponse>(),
        api.get("nodes").json<NodeListResponse>(),
    ]);

    return <Providers providers={providers} nodes={nodes} />;
};

export default ProvidersPage;
