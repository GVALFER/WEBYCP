import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { BackupArtifactListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Archives from "./archives";

const ArchivesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/backups/archives", searchParams);
    const archives = await api.get("backup-artifacts", { searchParams: query })
        .json<BackupArtifactListResponse>();

    await syncPage("/backups/archives", searchParams, query, archives.pagination);

    return <Archives archives={archives} />;
};

export default ArchivesPage;
