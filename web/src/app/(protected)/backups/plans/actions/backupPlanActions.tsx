"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Play, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { BackupPlanListResponse, BackupRunResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type BackupPlanActionsProps = {
    plan: BackupPlanListResponse["items"][number];
};

const BackupPlanActions = ({ plan }: BackupPlanActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (action: () => Promise<unknown>, success: string, usage = false) => {
        setPending(true);

        try {
            await action();
            await mutate((key) =>
                isPageKey(
                    key,
                    "backup-plans",
                    "backup-runs",
                    "backup-artifacts",
                    ...(usage ? ["accounts"] : []),
                    "jobs",
                ),
            );
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
            true,
        );

    return (
        <div className="flex items-center gap-2">
            <Button size="sm" variant="secondary" isPending={pending} onPress={() => void start()}>
                {pending ? <Spinner color="current" size="sm" /> : <Play className="size-4" />}
                Run now
            </Button>
            <Confirm
                title={`Delete ${plan.name}?`}
                description="The schedule will be removed. Existing backup archives will be retained."
                action="Delete plan"
                onConfirm={remove}
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
