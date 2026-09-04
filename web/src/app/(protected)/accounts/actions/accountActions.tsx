"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { Power, Trash2 } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { AccountJobResponse, AccountListResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type AccountActionsProps = {
    account: AccountListResponse["items"][number];
};

const AccountActions = ({ account }: AccountActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (enabled?: boolean) => {
        setPending(true);

        try {
            const path = `accounts/${encodeURIComponent(account.id)}`;

            if (enabled === undefined) {
                await api.delete(path).json<AccountJobResponse>();
            } else {
                await api.patch(path, { json: { enabled } }).json<AccountJobResponse>();
            }

            await mutate((key) => isPageKey(key, "accounts", "jobs"));
            toast.success(
                enabled === undefined
                    ? "Account queued for deletion"
                    : enabled
                      ? "Account enabled"
                      : "Account disabled",
            );
        } catch (error) {
            toast.danger("Account action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    const disabled = account.status === "pending" || pending;

    return (
        <div className="flex items-center gap-1">
            <Button
                isIconOnly
                size="sm"
                variant="tertiary"
                aria-label={account.enabled ? `Disable ${account.name}` : `Enable ${account.name}`}
                isPending={pending}
                isDisabled={account.status === "pending"}
                onPress={() => void run(!account.enabled)}
            >
                {pending ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <Power className="size-4" aria-hidden="true" />
                )}
            </Button>
            <Confirm
                title={`Delete ${account.name}?`}
                description="The account must be empty. Its home directory will be moved to the recovery trash."
                action="Delete account"
                onConfirm={() => run()}
            >
                <Button
                    isIconOnly
                    size="sm"
                    variant="danger-soft"
                    aria-label={`Delete ${account.name}`}
                    isDisabled={disabled}
                >
                    <Trash2 className="size-4" aria-hidden="true" />
                </Button>
            </Confirm>
        </div>
    );
};

export default AccountActions;
