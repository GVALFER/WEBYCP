import type { CertificateListResponse, DomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/utils/paginationServer";
import Certificates from "./certificates";

const CertificatesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/certificates", searchParams);
    const [certificates, domains] = await Promise.all([
        api.get("certificates", { searchParams: query }).json<CertificateListResponse>(),
        api
            .get("domains", { searchParams: { page: 1, size: 100 } })
            .json<DomainListResponse>(),
    ]);

    await syncPage("/certificates", searchParams, query, certificates.pagination);

    return <Certificates certificates={certificates} domains={domains} />;
};

export default CertificatesPage;
