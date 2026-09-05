"use client";

import type { ReactNode } from "react";
import { Table, type TableColumn } from "@/components/table/table";
import type { TableState } from "@/components/table/useTable";
import type { BackupArtifact, BackupArtifactListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";

type Props = {
    data?: BackupArtifactListResponse;
    pending: boolean;
    table: TableState;
    actions: (archive: BackupArtifact) => ReactNode;
};

const ArchiveTable = ({ data, pending, table, actions }: Props) => {
    const { dt } = useTimezone();

    const columns: TableColumn<BackupArtifact>[] = [
        {
            id: "archive",
            label: "Archive",
            isRowHeader: true,
            render: (archive) => (
                <div>
                    <div className="font-medium">{dt(archive.createdAt)}</div>
                    <div className="mt-1 font-mono text-xs text-foreground-400">{archive.id}</div>
                </div>
            ),
        },
        {
            id: "account",
            label: "Account ID",
            cellClassName: "font-mono text-xs text-foreground-500",
            render: (archive) => archive.accountId,
        },
        {
            id: "content",
            label: "Content",
            cellClassName: "text-foreground-500",
            render: ({ manifest }) => [
                manifest.files && "Files",
                manifest.databases && "Databases",
                manifest.metadata && "Metadata",
            ].filter(Boolean).join(" · "),
        },
        {
            id: "size",
            label: "Size",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (archive) => `${(archive.size / 1_048_576).toFixed(2)} MB`,
        },
        {
            id: "checksum",
            label: "SHA-256",
            cellClassName: "font-mono text-xs text-foreground-500",
            render: (archive) => <span title={archive.checksum}>{archive.checksum.slice(0, 12)}…</span>,
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: actions,
        },
    ];

    return <Table data={data} pending={pending} table={table} columns={columns} />;
};

export default ArchiveTable;
