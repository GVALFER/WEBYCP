import type { NodeListResponse, ServiceSettings } from "@/contracts/types";
import { api } from "@/lib/api";
import Services from "./services";

const ServicesPage = async () => {
    const [nodes, settings] = await Promise.all([
        api.get("nodes").json<NodeListResponse>(),
        api.get("service-settings").json<ServiceSettings>(),
    ]);

    return <Services nodes={nodes} settings={settings} />;
};

export default ServicesPage;
