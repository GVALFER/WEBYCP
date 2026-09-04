import type { CertificateListResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Certificates from "./certificates";

const CertificatesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/certificates", searchParams);

    const [certificates, websites] = await Promise.all([
        api
            .get("certificates", {
                searchParams: {
                    kind: "website",
                    ...query,
                },
            })
            .json<CertificateListResponse>(),
        api
            .get("websites", {
                searchParams: {
                    page: 1,
                    size: 100,
                },
            })
            .json<WebsiteListResponse>(),
    ]);

    await syncPage("/certificates", searchParams, query, certificates.pagination);

    return <Certificates certificates={certificates} websites={websites} />;
};

export default CertificatesPage;
