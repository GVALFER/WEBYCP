"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import { Confirm } from "@/components/actions/confirm";
import type { FTPAccount, FTPAccountResponse, Job } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

const FTPActions = ({ item }: { item: FTPAccount }) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();
    const path = `ftp-accounts/${encodeURIComponent(item.id)}`;

    const run = async (action: () => Promise<unknown>, message: string) => {
        setPending(true);
        try {
            await action();
            await mutate((key) => isPageKey(key, "ftp-accounts", "accounts", "jobs", "audit-events"));
            toast.success(message);
        } catch (error) {
            toast.danger("FTP action failed", { description: await errorMessage(error) });
        } finally {
            setPending(false);
        }
    };

    const toggle = () => run(
        () => api.patch(path, { json: { enabled: !item.enabled } }).json<FTPAccountResponse>(),
        item.enabled ? "FTP disable queued" : "FTP enable queued",
    );
    const remove = () => run(() => api.delete(path).json<Job>(), "FTP revocation queued");

    return (
        <div className="flex items-center gap-2">
            {!item.deleting && (
                <Confirm
                    title={`${item.enabled ? "Disable" : "Enable"} ${item.username}?`}
                    description="All FTP sessions for this hosting account will be disconnected. Other accounts are unaffected."
                    action={item.enabled ? "Disable" : "Enable"}
                    onConfirm={toggle}
                >
                    <Button
                        isIconOnly
                        size="sm"
                        variant="tertiary"
                        aria-label={`${item.enabled ? "Disable" : "Enable"} ${item.username}`}
                        isPending={pending}
                        isDisabled={item.status === "pending"}
                    >
                        {pending ? <Spinner color="current" size="sm" /> : <Power className="size-4" />}
                    </Button>
                </Confirm>
            )}
            <Confirm
                title={`Delete ${item.username}?`}
                description="Revokes this login and disconnects all FTP sessions for its hosting account. Customer files are not deleted."
                action={item.deleting ? "Retry deletion" : "Delete"}
                onConfirm={remove}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${item.username}`}
                    isDisabled={pending || item.status === "pending"}
                >
                    <Trash2 className="size-4" />
                </Button>
            </Confirm>
        </div>
    );
};

export default FTPActions;
