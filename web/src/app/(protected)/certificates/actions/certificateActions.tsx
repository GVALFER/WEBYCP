"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { CertificateJobResponse, CertificateListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

type CertificateActionsProps = {
    certificate: CertificateListResponse["items"][number];
};

type Action = "redirect" | "renew" | "";

const CertificateActions = ({ certificate }: CertificateActionsProps) => {
    const [pending, setPending] = useState<Action>("");
    const { mutate } = useSWRConfig();

    const run = async (key: Action, action: () => Promise<unknown>, success: string) => {
        setPending(key);

        try {
            await action();
            await Promise.all([mutate("certificates"), mutate("jobs")]);
            toast.success(success);
        } catch (error) {
            toast.danger("Certificate action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending("");
        }
    };

    const toggleRedirect = () =>
        run(
            "redirect",
            () =>
                api
                    .patch(`certificates/${encodeURIComponent(certificate.id)}`, {
                        json: { redirectHttps: !certificate.redirectHttps },
                    })
                    .json<CertificateJobResponse>(),
            certificate.redirectHttps ? "HTTPS redirect disabled" : "HTTPS redirect enabled",
        );

    const renew = () =>
        run(
            "renew",
            () =>
                api
                    .post(`certificates/${encodeURIComponent(certificate.id)}/renew`)
                    .json<CertificateJobResponse>(),
            "Certificate renewal queued",
        );

    return (
        <div className="flex items-center gap-2">
            {certificate.kind === "domain" && (
                <Button
                    size="sm"
                    variant="tertiary"
                    isPending={pending === "redirect"}
                    isDisabled={certificate.status === "pending" || pending === "renew"}
                    onPress={() => void toggleRedirect()}
                >
                    {pending === "redirect" ? (
                        <Spinner color="current" size="sm" />
                    ) : null}
                    {certificate.redirectHttps ? "Disable redirect" : "Enable redirect"}
                </Button>
            )}
            <Button
                isIconOnly
                size="sm"
                variant="secondary"
                aria-label={`Renew ${certificate.name}`}
                isPending={pending === "renew"}
                isDisabled={certificate.status === "pending" || pending === "redirect"}
                onPress={() => void renew()}
            >
                {pending === "renew" ? (
                    <Spinner color="current" size="sm" />
                ) : (
                    <RefreshCw className="size-4" />
                )}
            </Button>
        </div>
    );
};

export default CertificateActions;
