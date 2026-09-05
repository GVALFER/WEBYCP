"use client";

import useSWR from "swr";
import { useTable } from "@/components/table/useTable";
import type { BackupArtifactListResponse } from "@/contracts/types";
import ArchiveTable from "../components/archiveTable";
import RestoreBackup from "./actions/restoreBackup";

type Props = {
    archives: BackupArtifactListResponse;
};

const Restore = ({ archives }: Props) => {
    const table = useTable(archives.pagination);
    const { data, isLoading } = useSWR<BackupArtifactListResponse>(`backup-artifacts${table.query}`, {
        fallbackData: table.isInitialQuery ? archives : undefined,
    });

    return (
        <section className="panel-card overflow-hidden">
            <div className="px-6 py-5">
                <h2 className="text-base font-semibold">Restore an account backup</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Verify an archive, review its contents and choose what to restore.
                </div>
            </div>
            <ArchiveTable data={data} pending={isLoading} table={table} actions={(archive) => <RestoreBackup archive={archive} />} />
        </section>
    );
};

export default Restore;
