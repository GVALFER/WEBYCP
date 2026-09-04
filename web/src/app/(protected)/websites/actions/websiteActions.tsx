"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import type { WebsiteJobResponse, WebsiteListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type WebsiteActionsProps = {
    website: WebsiteListResponse["items"][number];
};

const WebsiteActions = ({ website }: WebsiteActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (enabled?: boolean) => {
        setPending(true);
        try {
            const path = `websites/${encodeURIComponent(website.id)}`;
            if (enabled === undefined) {
                await api.delete(path).json<WebsiteJobResponse>();
            } else {
                await api.patch(path, { json: { enabled } }).json<WebsiteJobResponse>();
            }
            await mutate((key) =>
                isPageKey(
                    key,
                    "websites",
                    "website-domains",
                    "certificates",
                    ...(enabled === undefined ? ["accounts"] : []),
                    "jobs",
                ),
            );
            toast.success(
                enabled === undefined
                    ? "Website queued for deletion"
                    : enabled
                      ? "Website enabled"
                      : "Website disabled",
            );
        } catch (error) {
            toast.danger("Website action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    const disabled = website.status === "pending" || pending;

    return (
        <div className="flex items-center gap-1">
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={website.enabled ? `Disable ${website.name}` : `Enable ${website.name}`}
                isPending={pending}
                isDisabled={website.status === "pending"}
                onPress={() => void run(!website.enabled)}
            >
                {pending ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <Power className="size-4" aria-hidden="true" />
                )}
            </Button>
            <Confirm
                title={`Delete ${website.name}?`}
                description="The document root will be moved to the recovery trash."
                action="Delete website"
                onConfirm={() => run()}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${website.name}`}
                    isDisabled={disabled}
                >
                    <Trash2 className="size-4" aria-hidden="true" />
                </Button>
            </Confirm>
        </div>
    );
};

export default WebsiteActions;
