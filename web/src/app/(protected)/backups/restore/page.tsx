import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { BackupArtifactListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Restore from "./restore";

const RestorePage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/backups/restore", searchParams);
    const archives = await api.get("backup-artifacts", { searchParams: query })
        .json<BackupArtifactListResponse>();

    await syncPage("/backups/restore", searchParams, query, archives.pagination);

    return <Restore archives={archives} />;
};

export default RestorePage;
