import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { WebsiteDomainListResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Domains from "./domains";

const DomainsPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/domains", searchParams);

    const [domains, websites] = await Promise.all([
        api
            .get("website-domains", { searchParams: { kind: "primary", ...query } })
            .json<WebsiteDomainListResponse>(),
        api.get("websites", { searchParams: { page: 1, size: 100 } }).json<WebsiteListResponse>(),
    ]);

    await syncPage("/domains", searchParams, query, domains.pagination);

    return <Domains domains={domains} websites={websites} />;
};

export default DomainsPage;
