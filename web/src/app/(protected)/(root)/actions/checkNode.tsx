"use client";

import { Button, toast } from "@heroui/react";
import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { Job, NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { cn } from "@/utils/classnames";
import { errorMessage } from "@/utils/errors";

type CheckNodeProps = {
    node: NodeListResponse["items"][number];
};

const CheckNode = ({ node }: CheckNodeProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const check = async () => {
        setPending(true);

        try {
            await api.post(`nodes/${encodeURIComponent(node.id)}/probe`).json<Job>();
            await Promise.all([mutate("nodes"), mutate("jobs")]);
            toast.success("Agent check completed");
        } catch (error) {
            toast.danger("Agent check failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    return (
        <Button
            size="sm"
            variant="secondary"
            isDisabled={pending}
            onPress={() => void check()}
        >
            <RefreshCw className={cn("size-4", pending && "animate-spin")} />
            Check agent
        </Button>
    );
};

export default CheckNode;
