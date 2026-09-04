"use client";

import { Button } from "@heroui/react";
import { Trash2 } from "lucide-react";
import type { DatabaseGrantJobResponse, DatabaseGrantListResponse } from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { useDatabaseAction } from "./useDatabaseAction";

type GrantActionsProps = {
    grant: DatabaseGrantListResponse["items"][number];
    database: string;
    user: string;
};

const GrantActions = ({ grant, database, user }: GrantActionsProps) => {
    const { pending, run } = useDatabaseAction();

    const remove = () => {
        const path = `databases/${encodeURIComponent(grant.databaseId)}/users/${encodeURIComponent(grant.databaseUserId)}`;
        return run(() => api.delete(path).json<DatabaseGrantJobResponse>(), "Access revoked");
    };

    return (
        <Confirm
            title="Revoke database access?"
            description={`${user || "This user"} will no longer have access to ${database || "this database"}.`}
            action="Revoke access"
            onConfirm={() => void remove()}
        >
            <Button
                isIconOnly
                size="sm"
                variant="danger-soft"
                aria-label="Revoke grant"
                isDisabled={pending}
            >
                <Trash2 className="size-4" />
            </Button>
        </Confirm>
    );
};

export default GrantActions;
