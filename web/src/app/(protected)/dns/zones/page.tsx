import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type {
    AccountListResponse,
    DNSProviderListResponse,
    DNSSettings,
    DNSZoneListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import Zones from "./zones";

const ZonesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/dns/zones", searchParams);
    const [zones, accounts, providers, settings] = await Promise.all([
        api.get("dns/zones", { searchParams: query }).json<DNSZoneListResponse>(),
        api
            .get("accounts", { searchParams: { page: 1, size: 100 } })
            .json<AccountListResponse>(),
        api.get("dns/providers").json<DNSProviderListResponse>(),
        api.get("dns/settings").json<DNSSettings>(),
    ]);

    await syncPage("/dns/zones", searchParams, query, zones.pagination);

    return (
        <Zones zones={zones} accounts={accounts} providers={providers} settings={settings} />
    );
};

export default ZonesPage;
