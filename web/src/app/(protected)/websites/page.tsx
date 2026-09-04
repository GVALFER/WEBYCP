import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type {
    AccountListResponse,
    ServiceSettings,
    WebsiteDomainListResponse,
    WebsiteListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import Websites from "./websites";

const WebsitesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/websites", searchParams);

    const [accounts, websites, domains, settings] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("websites", { searchParams: query }).json<WebsiteListResponse>(),
        api
            .get("website-domains", {
                searchParams: { kind: "primary", page: 1, size: 100 },
            })
            .json<WebsiteDomainListResponse>(),
        api.get("service-settings").json<ServiceSettings>(),
    ]);

    await syncPage("/websites", searchParams, query, websites.pagination);

    return (
        <Websites accounts={accounts} websites={websites} domains={domains} settings={settings} />
    );
};

export default WebsitesPage;
