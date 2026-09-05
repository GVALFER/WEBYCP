import type { AuditEventListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Audit from "./audit";

const AuditPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/audit", searchParams);

    const raw = (await searchParams).jobId;
    const jobId = typeof raw === "string" ? raw : "";

    const events = await api
        .get("audit-events", { searchParams: { ...query, jobId } })
        .json<AuditEventListResponse>();

    await syncPage("/audit", searchParams, query, events.pagination);

    return <Audit events={events} jobId={jobId} />;
};

export default AuditPage;
