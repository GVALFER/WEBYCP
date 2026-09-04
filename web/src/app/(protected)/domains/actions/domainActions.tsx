"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Pencil, Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { DomainJobResponse, DomainListResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { TextDialog } from "@/components/actions/textDialog";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

type DomainActionsProps = {
    domain: DomainListResponse["items"][number];
};

const DomainActions = ({ domain }: DomainActionsProps) => {
    const [edit, setEdit] = useState(false);
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (action: () => Promise<unknown>, success: string) => {
        setPending(true);

        try {
            await action();
            await mutate((key) => isPageKey(key, "domains", "jobs"));
            toast.success(success);
        } catch (error) {
            toast.danger("Domain action failed", {
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
                    .patch(`domains/${encodeURIComponent(domain.id)}`, {
                        json: { enabled: !domain.enabled },
                    })
                    .json<DomainJobResponse>(),
            domain.enabled ? "Domain disabled" : "Domain enabled",
        );

    const remove = () =>
        run(
            () => api.delete(`domains/${encodeURIComponent(domain.id)}`).json<DomainJobResponse>(),
            "Domain queued for deletion",
        );

    const rename = async (name: string) => {
        if (name === domain.name) {
            setEdit(false);
            return;
        }

        await run(
            () =>
                api
                    .patch(`domains/${encodeURIComponent(domain.id)}`, {
                        json: { name },
                    })
                    .json<DomainJobResponse>(),
            "Domain rename queued",
        );
        setEdit(false);
    };

    const disabled = domain.status === "pending" || pending;

    return (
        <>
            <div className="flex items-center gap-1">
                <Button
                    isIconOnly
                    size="sm"
                    variant="tertiary"
                    aria-label={`Rename ${domain.name}`}
                    isDisabled={domain.status !== "active" || pending}
                    onPress={() => setEdit(true)}
                >
                    <Pencil className="size-4" aria-hidden="true" />
                </Button>
                <Button
                    isIconOnly
                    size="sm"
                    variant="tertiary"
                    aria-label={domain.enabled ? `Disable ${domain.name}` : `Enable ${domain.name}`}
                    isPending={pending}
                    isDisabled={domain.status === "pending"}
                    onPress={() => void toggle()}
                >
                    {pending ? (
                        <Spinner color="current" size="sm" />
                    ) : (
                        <Power className="size-4" aria-hidden="true" />
                    )}
                </Button>
                <Confirm
                    title={`Delete ${domain.name}?`}
                    description="Its document root will be moved to the recovery trash."
                    action="Delete domain"
                    onConfirm={remove}
                >
                    <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${domain.name}`}
                        isDisabled={disabled}
                    >
                        <Trash2 className="size-4" aria-hidden="true" />
                    </Button>
                </Confirm>
            </div>
            <TextDialog
                open={edit}
                title="Rename domain"
                label="Hostname"
                value={domain.name}
                schema={domainField}
                pending={pending}
                onOpenChange={setEdit}
                onSubmit={rename}
            />
        </>
    );
};

export default DomainActions;
