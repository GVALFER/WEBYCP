"use client";

import { Button } from "@heroui/react";
import { Trash2 } from "lucide-react";
import type {
    DatabaseJobResponse,
    DatabaseListResponse,
    DatabaseUserJobResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { Confirm } from "@/components/actions/confirm";
import { api } from "@/lib/api";
import { useDatabaseAction } from "./useDatabaseAction";

type Resource =
    | DatabaseListResponse["items"][number]
    | DatabaseUserListResponse["items"][number];

type ResourceActionsProps = {
    kind: "database" | "user";
    resource: Resource;
};

const ResourceActions = ({ kind, resource }: ResourceActionsProps) => {
    const { pending, run } = useDatabaseAction();

    const remove = () =>
        kind === "database"
            ? run(
                  () =>
                      api
                          .delete(`databases/${encodeURIComponent(resource.id)}`)
                          .json<DatabaseJobResponse>(),
                  "Database queued for deletion",
              )
            : run(
                  () =>
                      api
                          .delete(`database-users/${encodeURIComponent(resource.id)}`)
                          .json<DatabaseUserJobResponse>(),
                  "Database user queued for deletion",
              );

    const description =
        kind === "database"
            ? `Delete ${resource.name}? Its data will be permanently removed.`
            : `Delete database user ${resource.name}?`;

    return (
        <Confirm
            title={`Delete ${resource.name}?`}
            description={description}
            action="Delete"
            onConfirm={() => void remove()}
        >
            <Button
                isIconOnly
                size="sm"
                variant="danger-soft"
                aria-label={`Delete ${resource.name}`}
                isDisabled={pending || resource.status === "pending"}
            >
                <Trash2 className="size-4" />
            </Button>
        </Confirm>
    );
};

export default ResourceActions;
