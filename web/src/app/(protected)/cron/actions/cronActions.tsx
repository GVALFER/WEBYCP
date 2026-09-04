"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { CronJobListResponse, CronJobResponse, Job } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

type CronActionsProps = {
    item: CronJobListResponse["items"][number];
};

const CronActions = ({ item }: CronActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();
    const path = `cron-jobs/${encodeURIComponent(item.id)}`;

    const run = async (action: () => Promise<unknown>, success: string) => {
        setPending(true);

        try {
            await action();
            await Promise.all([mutate("cron-jobs"), mutate("jobs")]);
            toast.success(success);
        } catch (error) {
            toast.danger("Action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    const toggle = () =>
        run(
            () =>
                api
                    .patch(path, {
                        json: {
                            accountId: item.accountId,
                            name: item.name,
                            schedule: item.schedule,
                            command: item.command,
                            enabled: !item.enabled,
                        },
                    })
                    .json<CronJobResponse>(),
            item.enabled ? "Cron job disabled" : "Cron job enabled",
        );

    const remove = () =>
        run(() => api.delete(path).json<Job>(), "Cron job queued for deletion");

    return (
        <div className="flex items-center gap-2">
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={item.enabled ? `Disable ${item.name}` : `Enable ${item.name}`}
                isPending={pending}
                isDisabled={item.status === "pending"}
                onPress={() => void toggle()}
            >
                {pending ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <Power className="size-4" />
                )}
            </Button>
            <Confirm
                title={`Delete ${item.name}?`}
                description="The schedule will be removed from the hosting account."
                action="Delete"
                onConfirm={remove}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${item.name}`}
                    isDisabled={pending}
                >
                    <Trash2 className="size-4" />
                </Button>
            </Confirm>
        </div>
    );
};

export default CronActions;
