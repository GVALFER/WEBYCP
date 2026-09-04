"use client";

import { Button, toast } from "@heroui/react";
import { Play, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { BackupPlanListResponse, BackupRunResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

type BackupPlanActionsProps = {
    plan: BackupPlanListResponse["items"][number];
};

const BackupPlanActions = ({ plan }: BackupPlanActionsProps) => {
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

    const start = () =>
        run(
            () =>
                api
                    .post(`backup-plans/${encodeURIComponent(plan.id)}/runs`)
                    .json<BackupRunResponse>(),
            "Backup queued",
        );

    const remove = () =>
        run(
            async () => {
                await api.delete(`backup-plans/${encodeURIComponent(plan.id)}`);
            },
            "Backup plan deleted",
        );

    return (
        <div className="flex items-center gap-2">
            <Button
                size="sm"
                variant="secondary"
                isDisabled={pending}
                onPress={() => void start()}
            >
                <Play className="size-4" />
                Run now
            </Button>
            <Confirm
                title={`Delete ${plan.name}?`}
                description="The schedule will be removed. Existing backup artifacts will be retained."
                action="Delete plan"
                onConfirm={() => void remove()}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${plan.name}`}
                    isDisabled={pending}
                >
                    <Trash2 className="size-4" />
                </Button>
            </Confirm>
        </div>
    );
};

export default BackupPlanActions;
