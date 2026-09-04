"use client";

import { LockKeyhole } from "lucide-react";
import useSWR from "swr";
import type { CertificateListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CreatePanelCertificate from "./actions/createPanelCertificate";
import PanelCertificateActions from "./actions/panelCertificateActions";

const Settings = ({ certificates }: { certificates: CertificateListResponse }) => {
    const { dt } = useTimezone();

    const { data } = useSWR<CertificateListResponse>("certificates?kind=panel&page=1&size=10", {
        fallbackData: certificates,
    });

    const certificate = data?.items[0];

    return (
        <section className="panel-card overflow-hidden">
            <div className="border-b border-divider px-6 py-5">
                <h2 className="text-base font-semibold">Panel TLS</h2>
                <div className="mt-1 text-sm text-foreground-500">
                    Certificate used by the WEBYCP control panel itself.
                </div>
            </div>
            <div className="flex flex-col gap-5 px-6 py-6 sm:flex-row sm:items-center sm:justify-between">
                {certificate ? (
                    <div className="flex min-w-0 items-center gap-4">
                        <div className="icon-box">
                            <LockKeyhole className="size-5" aria-hidden="true" />
                        </div>
                        <div className="min-w-0">
                            <div className="flex items-center gap-2">
                                <span className="truncate font-medium">{certificate.name}</span>
                                <span
                                    className={cn(
                                        "rounded-full px-2.5 py-1 text-xs capitalize",
                                        statusClass(certificate.status),
                                    )}
                                >
                                    {certificate.status}
                                </span>
                            </div>
                            <div className="mt-1 text-xs text-foreground-400">
                                {certificate.expiresAt
                                    ? `Expires ${dt(certificate.expiresAt)}`
                                    : "Certificate request pending"}
                            </div>
                        </div>
                    </div>
                ) : (
                    <div className="text-sm text-foreground-500">
                        No panel certificate has been configured.
                    </div>
                )}
                <div className="flex items-center gap-2">
                    {certificate ? <PanelCertificateActions certificate={certificate} /> : null}
                    <CreatePanelCertificate />
                </div>
            </div>
        </section>
    );
};

export default Settings;
