"use client";

import { LockKeyhole } from "lucide-react";
import useSWR from "swr";
import type { CertificateListResponse, DomainListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { useSession } from "@/providers/SessionProvider";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CertificateActions from "./actions/certificateActions";
import CreateCertificate from "./actions/createCertificate";
import CreatePanelCertificate from "./actions/createPanelCertificate";

type CertificatesProps = {
    certificates: CertificateListResponse;
    domains: DomainListResponse;
};

const Certificates = ({ certificates, domains }: CertificatesProps) => {
    const session = useSession();
    const { dt } = useTimezone();

    const { data } = useSWR<CertificateListResponse>("certificates", {
        fallbackData: certificates,
    });

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">TLS certificates</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Let&apos;s Encrypt certificates, expiry and HTTPS redirect policy.
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <CreateCertificate domains={domains} email={session.user.email} />
                    {session.user.role === "admin" && (
                        <CreatePanelCertificate email={session.user.email} />
                    )}
                </div>
            </div>
            <div className="divide-y divide-divider">
                {data?.items.length ? (
                    data.items.map((certificate) => (
                        <div key={certificate.id} className="px-6 py-5">
                            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                                <div className="flex min-w-0 items-center gap-4">
                                    <div className="icon-box">
                                        <LockKeyhole className="size-5" />
                                    </div>
                                    <div className="min-w-0">
                                        <div className="truncate font-medium">
                                            {certificate.name}
                                        </div>
                                        <div className="mt-1 text-xs text-foreground-400">
                                            {certificate.kind} · expires{" "}
                                            {certificate.expiresAt
                                                ? dt(certificate.expiresAt)
                                                : "Never"}
                                        </div>
                                    </div>
                                </div>
                                <div className="flex flex-wrap items-center gap-2">
                                    <span
                                        className={cn(
                                            "rounded-full px-2.5 py-1 text-xs capitalize",
                                            statusClass(certificate.status),
                                        )}
                                    >
                                        {certificate.status}
                                    </span>
                                    <CertificateActions certificate={certificate} />
                                </div>
                            </div>
                            {certificate.names.length > 1 && (
                                <div className="mt-3 text-xs text-foreground-500">
                                    SANs: {certificate.names.join(", ")}
                                </div>
                            )}
                            {certificate.error && (
                                <div className="mt-3 text-sm text-danger">
                                    {certificate.error}
                                </div>
                            )}
                        </div>
                    ))
                ) : (
                    <div className="px-6 py-12 text-center text-sm text-foreground-400">
                        No certificates yet. HTTP remains available for ACME bootstrap.
                    </div>
                )}
            </div>
        </section>
    );
};

export default Certificates;
