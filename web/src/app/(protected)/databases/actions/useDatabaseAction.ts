"use client";

import { toast } from "@heroui/react";
import { useCallback, useState } from "react";
import { useSWRConfig } from "swr";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

export const useDatabaseAction = () => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = useCallback(
        async <T>(action: () => Promise<T>, success: string, usage = false) => {
            setPending(true);

            try {
                const response = await action();
                await mutate((key) =>
                    isPageKey(
                        key,
                        "databases",
                        "database-users",
                        "database-grants",
                        ...(usage ? ["accounts"] : []),
                        "jobs",
                    ),
                );
                toast.success(success);
                return response;
            } catch (error) {
                toast.danger("Action failed", {
                    description: await errorMessage(error),
                });
            } finally {
                setPending(false);
            }
        },
        [mutate],
    );

    return { pending, run };
};
