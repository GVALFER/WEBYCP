"use client";

import { Button, Spinner, toast } from "@heroui/react";
import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { useSWRConfig } from "swr";
import type { CertificateJobResponse, CertificateListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { errorMessage } from "@/utils/errors";
import { isPageKey } from "@/utils/pagination";

type PanelCertificateActionsProps = {
    certificate: CertificateListResponse["items"][number];
};

const PanelCertificateActions = ({ certificate }: PanelCertificateActionsProps) => {
    const [pending, setPending] = useState(false);
    const { mutate } = useSWRConfig();

    const renew = async () => {
        setPending(true);

        try {
            await api
                .post(`certificates/${encodeURIComponent(certificate.id)}/renew`)
                .json<CertificateJobResponse>();
            await mutate((key) => isPageKey(key, "certificates", "jobs"));
            toast.success("Panel certificate renewal queued");
        } catch (error) {
            toast.danger("Certificate action failed", {
                description: await errorMessage(error),
            });
        } finally {
            setPending(false);
        }
    };

    return (
        <Button
            isIconOnly
            size="sm"
            variant="secondary"
            aria-label={`Renew ${certificate.name}`}
            isPending={pending}
            isDisabled={certificate.status === "pending"}
            onPress={() => void renew()}
        >
            {pending ? <Spinner color="current" size="sm" /> : <RefreshCw className="size-4" />}
        </Button>
    );
};

export default PanelCertificateActions;
