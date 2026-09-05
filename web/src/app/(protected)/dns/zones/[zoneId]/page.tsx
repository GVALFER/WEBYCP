import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { DNSRecordListResponse, DNSSettings, DNSZone } from "@/contracts/types";
import { api } from "@/lib/api";
import Records from "./records";

type Props = PageProps & {
    params: Promise<{ zoneId: string }>;
};

const RecordsPage = async ({ params, searchParams }: Props) => {
    const { zoneId } = await params;
    const path = `/dns/zones/${encodeURIComponent(zoneId)}`;
    const query = await getPageQuery(path, searchParams);
    const apiPath = `dns/zones/${encodeURIComponent(zoneId)}`;
    const [zone, records, settings] = await Promise.all([
        api.get(apiPath).json<DNSZone>(),
        api
            .get(`${apiPath}/records`, { searchParams: query })
            .json<DNSRecordListResponse>(),
        api.get("dns/settings").json<DNSSettings>(),
    ]);

    await syncPage(path, searchParams, query, records.pagination);

    return <Records zone={zone} records={records} defaultTtl={settings.defaultTtl} />;
};

export default RecordsPage;
