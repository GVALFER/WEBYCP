"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Pencil } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { TextDialog } from "@/components/actions/textDialog";
import type { WebsiteDomainJobResponse, WebsiteDomainListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";
import { domainField } from "@/utils/validation";

type DomainActionsProps = {
    domain: WebsiteDomainListResponse["items"][number];
};

const DomainActions = ({ domain }: DomainActionsProps) => {
    const [open, setOpen] = useState(false);
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const rename = async (hostname: string) => {
        setPending(true);
        try {
            await api
                .patch(`website-domains/${encodeURIComponent(domain.id)}`, {
                    json: { hostname },
                })
                .json<WebsiteDomainJobResponse>();
            await mutate((key) => isPageKey(key, "website-domains", "certificates", "jobs"));
            setOpen(false);
            toast.success("Primary domain rename queued");
        } catch (error) {
            toast.danger("Domain action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    return (
        <>
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={`Rename ${domain.hostname}`}
                isPending={pending}
                isDisabled={domain.status === "pending"}
                onPress={() => setOpen(true)}
            >
                {pending ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <Pencil className="size-4" aria-hidden="true" />
                )}
            </Button>
            <TextDialog
                open={open}
                title="Rename primary domain"
                label="Hostname"
                value={domain.hostname}
                schema={domainField}
                pending={pending}
                onOpenChange={setOpen}
                onSubmit={rename}
            />
        </>
    );
};

export default DomainActions;
