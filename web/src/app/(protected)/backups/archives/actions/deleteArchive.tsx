"use client";

import { Button, toast } from "@heroui/react";
import { Trash2 } from "lucide-react";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import type { BackupArtifact } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type Props = {
    archive: BackupArtifact;
};

const DeleteArchive = ({ archive }: Props) => {
    const { mutate } = useSWRConfig();

    const remove = async () => {
        try {
            await api.delete(`backup-artifacts/${encodeURIComponent(archive.id)}`);
            await mutate((key) => isPageKey(key, "backup-artifacts"));
            toast.success("Backup archive deleted");
        } catch (error) {
            toast.danger("Backup deletion failed", { description: await errorMessage(error) });
        }
    };

    return (
        <Confirm
            title="Delete this backup?"
            description="The backup archive will be permanently deleted from its server."
            action="Delete backup"
            onConfirm={remove}
        >
            <Button isIconOnly size="sm" variant="danger-soft" aria-label={`Delete archive ${archive.id}`}>
                <Trash2 className="size-4" />
            </Button>
        </Confirm>
    );
};

export default DeleteArchive;
