"use client";

import { Button, toast } from "@heroui/react";
import { ArchiveRestore, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type {
    BackupArtifactListResponse,
    BackupManifest,
    Job,
} from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

type BackupArtifactActionsProps = {
    artifact: BackupArtifactListResponse["items"][number];
};

const BackupArtifactActions = ({ artifact }: BackupArtifactActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (action: () => Promise<unknown>, success: string) => {
        setPending(true);

        try {
            await action();
            await Promise.all([
                mutate("backup-plans"),
                mutate("backup-runs"),
                mutate("backup-artifacts"),
                mutate("jobs"),
            ]);
            toast.success(success);
        } catch (error) {
            toast.danger("Backup action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    const restore = () =>
        run(async () => {
            const path = `backup-artifacts/${encodeURIComponent(artifact.id)}/restore`;
            const manifest = await api.get(path).json<BackupManifest>();

            await api
                .post(path, {
                    json: {
                        files: manifest.files,
                        databases: manifest.databases,
                        metadata: manifest.metadata,
                    },
                })
                .json<Job>();
        }, "Restore queued");

    const remove = () =>
        run(
            async () => {
                await api.delete(`backup-artifacts/${encodeURIComponent(artifact.id)}`);
            },
            "Backup artifact deleted",
        );

    return (
        <div className="flex items-center gap-2">
            <Confirm
                title="Restore this backup?"
                description="The verified manifest will be restored. Existing files and databases may be overwritten."
                action="Restore backup"
                onConfirm={() => void restore()}
            >
                <Button size="sm" variant="secondary" isDisabled={pending}>
                    <ArchiveRestore className="size-4" />
                    Restore
                </Button>
            </Confirm>
            <Confirm
                title="Delete this backup?"
                description="The local backup artifact will be permanently deleted."
                action="Delete backup"
                onConfirm={() => void remove()}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label="Delete backup artifact"
                    isDisabled={pending}
                >
                    <Trash2 className="size-4" />
                </Button>
            </Confirm>
        </div>
    );
};

export default BackupArtifactActions;
