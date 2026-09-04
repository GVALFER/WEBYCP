import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { WebsiteDomainListResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Aliases from "./aliases";

const AliasesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/aliases", searchParams);

    const [aliases, websites] = await Promise.all([
        api
            .get("website-domains", {
                searchParams: {
                    kind: "alias",
                    ...query,
                },
            })
            .json<WebsiteDomainListResponse>(),
        api
            .get("websites", {
                searchParams: {
                    page: 1,
                    size: 100,
                },
            })
            .json<WebsiteListResponse>(),
    ]);

    await syncPage("/aliases", searchParams, query, aliases.pagination);

    return <Aliases aliases={aliases} websites={websites} />;
};

export default AliasesPage;
