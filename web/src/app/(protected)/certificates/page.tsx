import type { CertificateListResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Certificates from "./certificates";

const CertificatesPage = async () => {
    const [certificates, domains] = await Promise.all([
        api.get("certificates").json<CertificateListResponse>(),
        api.get("domains").json<DomainListResponse>(),
    ]);

    return <Certificates certificates={certificates} domains={domains} />;
};

export default CertificatesPage;
