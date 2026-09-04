"use client";

import { Button, toast } from "@heroui/react";
import { Pencil, Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { DomainAliasJobResponse, DomainAliasListResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { TextDialog } from "@/components/actions/textDialog";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { domainField } from "@/utils/validation";

type AliasActionsProps = {
    alias: DomainAliasListResponse["items"][number];
    domainId: string;
};

const AliasActions = ({ alias, domainId }: AliasActionsProps) => {
    const [edit, setEdit] = useState(false);
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();
    const aliasesKey = `domains/${encodeURIComponent(domainId)}/aliases`;
    const path = `${aliasesKey}/${encodeURIComponent(alias.id)}`;

    const run = async (action: () => Promise<unknown>, success: string) => {
        setPending(true);

        try {
            await action();
            await Promise.all([mutate(aliasesKey), mutate("domains"), mutate("jobs")]);
            toast.success(success);
        } catch (error) {
            toast.danger("Alias action failed", {
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
                    .patch(path, { json: { enabled: !alias.enabled } })
                    .json<DomainAliasJobResponse>(),
            alias.enabled ? "Alias disabled" : "Alias enabled",
        );

    const remove = () =>
        run(
            () => api.delete(path).json<DomainAliasJobResponse>(),
            "Alias queued for deletion",
        );

    const rename = async (name: string) => {
        if (name === alias.name) {
            setEdit(false);
            return;
        }

        await run(
            () =>
                api
                    .patch(path, { json: { name } })
                    .json<DomainAliasJobResponse>(),
            "Alias rename queued",
        );
        setEdit(false);
    };

    const disabled = alias.status === "pending" || pending;

    return (
        <>
            <div className="flex items-center gap-1">
                <Button
                    isIconOnly
                    size="sm"
                    variant="tertiary"
                    aria-label={`Rename ${alias.name}`}
                    isDisabled={disabled}
                    onPress={() => setEdit(true)}
                >
                    <Pencil className="size-4" aria-hidden="true" />
                </Button>
                <Button
                    isIconOnly
                    size="sm"
                    variant="tertiary"
                    aria-label={alias.enabled ? `Disable ${alias.name}` : `Enable ${alias.name}`}
                    isDisabled={disabled}
                    onPress={() => void toggle()}
                >
                    <Power className="size-4" aria-hidden="true" />
                </Button>
                <Confirm
                    title={`Delete ${alias.name}?`}
                    description="This hostname will stop serving the primary domain."
                    action="Delete alias"
                    onConfirm={() => void remove()}
                >
                    <Button
                        isIconOnly
                        size="sm"
                        variant="danger-soft"
                        aria-label={`Delete ${alias.name}`}
                        isDisabled={disabled}
                    >
                        <Trash2 className="size-4" aria-hidden="true" />
                    </Button>
                </Confirm>
            </div>
            <TextDialog
                open={edit}
                title="Rename alias"
                label="Hostname"
                value={alias.name}
                schema={domainField}
                pending={pending}
                onOpenChange={setEdit}
                onSubmit={rename}
            />
        </>
    );
};

export default AliasActions;
