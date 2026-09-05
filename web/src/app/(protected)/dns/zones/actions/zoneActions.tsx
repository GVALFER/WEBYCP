"use client";

import { Button, toast } from "@heroui/react";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import type { DNSZone, DNSZoneJobResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

const ZoneActions = ({ zone }: { zone: DNSZone }) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const remove = async () => {
        setPending(true);
        try {
            await api.delete(`dns/zones/${encodeURIComponent(zone.id)}`).json<DNSZoneJobResponse>();
            await mutate((key) => isPageKey(key, "dns/zones", "jobs"));
            toast.success("DNS zone queued for deletion");
        } catch (error) {
            toast.danger("DNS zone could not be deleted", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    return (
        <Confirm
            title={`Delete ${zone.name}?`}
            description="The authoritative zone and all records managed by WEBYCP will be removed."
            action="Delete zone"
            onConfirm={remove}
        >
            <Button
                isIconOnly
                size="sm"
                variant="danger-soft"
                aria-label={`Delete ${zone.name}`}
                isDisabled={zone.status === "pending" || zone.status === "deleting" || pending}
            >
                <Trash2 className="size-4" aria-hidden="true" />
            </Button>
        </Confirm>
    );
};

export default ZoneActions;
