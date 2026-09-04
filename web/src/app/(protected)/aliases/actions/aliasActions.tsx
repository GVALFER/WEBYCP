"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Pencil, Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import { TextDialog } from "@/components/actions/textDialog";
import type { WebsiteDomainJobResponse, WebsiteDomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

type Action = "rename" | "toggle" | "delete" | "";

type AliasActionsProps = {
    alias: WebsiteDomainListResponse["items"][number];
};

const AliasActions = ({ alias }: AliasActionsProps) => {
    const [action, setAction] = useState<Action>("");
    const [renameOpen, setRenameOpen] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (next: Action, json?: { hostname?: string; enabled?: boolean }) => {
        setAction(next);
        try {
            const path = `website-domains/${encodeURIComponent(alias.id)}`;
            if (next === "delete") {
                await api.delete(path).json<WebsiteDomainJobResponse>();
            } else {
                await api.patch(path, { json }).json<WebsiteDomainJobResponse>();
            }
            await mutate((key) =>
                isPageKey(
                    key,
                    "website-domains",
                    "certificates",
                    ...(next === "delete" ? ["accounts"] : []),
                    "jobs",
                ),
            );
            setRenameOpen(false);
            toast.success(
                next === "rename"
                    ? "Alias rename queued"
                    : next === "delete"
                      ? "Alias deletion queued"
                      : alias.enabled
                        ? "Alias disabled"
                        : "Alias enabled",
            );
        } catch (error) {
            toast.danger("Domain action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setAction("");
        }
    };

    const disabled = alias.status === "pending" || action !== "";

    return (
        <div className="flex items-center gap-1">
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={`Rename ${alias.hostname}`}
                isDisabled={disabled}
                onPress={() => setRenameOpen(true)}
            >
                <Pencil className="size-4" aria-hidden="true" />
            </Button>
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={
                    alias.enabled ? `Disable ${alias.hostname}` : `Enable ${alias.hostname}`
                }
                isPending={action === "toggle"}
                isDisabled={
                    alias.status === "pending" || action === "delete" || action === "rename"
                }
                onPress={() => void run("toggle", { enabled: !alias.enabled })}
            >
                {action === "toggle" ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <Power className="size-4" aria-hidden="true" />
                )}
            </Button>
            <Confirm
                title={`Delete ${alias.hostname}?`}
                description="The hostname will stop serving this website."
                action="Delete alias"
                onConfirm={() => run("delete")}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${alias.hostname}`}
                    isDisabled={disabled}
                >
                    <Trash2 className="size-4" aria-hidden="true" />
                </Button>
            </Confirm>
            <TextDialog
                open={renameOpen}
                title="Rename alias"
                label="Hostname"
                value={alias.hostname}
                schema={domainField}
                pending={action === "rename"}
                onOpenChange={setRenameOpen}
                onSubmit={(hostname) => run("rename", { hostname })}
            />
        </div>
    );
};

export default AliasActions;
