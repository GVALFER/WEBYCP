"use client";

import { Button, toast } from "@heroui/react";
import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { CertificateJobResponse, CertificateListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";

type CertificateActionsProps = {
    certificate: CertificateListResponse["items"][number];
};

const CertificateActions = ({ certificate }: CertificateActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const run = async (action: () => Promise<unknown>, success: string) => {
        setPending(true);

        try {
            await action();
            await Promise.all([mutate("certificates"), mutate("jobs")]);
            toast.success(success);
        } catch (error) {
            toast.danger("Certificate action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    const toggleRedirect = () =>
        run(
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
            () =>
                api
                    .post(`certificates/${encodeURIComponent(certificate.id)}/renew`)
                    .json<CertificateJobResponse>(),
            "Certificate renewal queued",
        );

    const disabled = certificate.status === "pending" || pending;

    return (
        <div className="flex items-center gap-2">
            {certificate.kind === "domain" && (
                <Button
                    size="sm"
                    variant="tertiary"
                    isDisabled={disabled}
                    onPress={() => void toggleRedirect()}
                >
                    {certificate.redirectHttps ? "Disable redirect" : "Enable redirect"}
                </Button>
            )}
            <Button
                isIconOnly
                size="sm"
                variant="secondary"
                aria-label={`Renew ${certificate.name}`}
                isDisabled={disabled}
                onPress={() => void renew()}
            >
                <RefreshCw className="size-4" />
            </Button>
        </div>
    );
};

export default CertificateActions;
