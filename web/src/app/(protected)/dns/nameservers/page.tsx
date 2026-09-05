import type { DNSSettings } from "@/contracts/types";
import { api } from "@/lib/api";
import Nameservers from "./nameservers";

const NameserversPage = async () => {
    const settings = await api.get("dns/settings").json<DNSSettings>();
    return <Nameservers settings={settings} />;
};

export default NameserversPage;
