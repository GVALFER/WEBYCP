"use client";

import { toast } from "@heroui/react";
import { useCallback, useState } from "react";
import { useSWRConfig } from "swr";
import { errorMessage } from "@/utils/errors";

export const useDatabaseAction = () => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = useCallback(
        async <T,>(action: () => Promise<T>, success: string) => {
            setPending(true);

            try {
                const response = await action();
                await Promise.all([
                    mutate("databases"),
                    mutate("database-users"),
                    mutate("database-grants"),
                    mutate("jobs"),
                ]);
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
