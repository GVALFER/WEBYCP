"use client";

import { LockKeyhole } from "lucide-react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type { CertificateListResponse, WebsiteListResponse } from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import CertificateActions from "./actions/certificateActions";
import CreateCertificate from "./actions/createCertificate";

type CertificatesProps = {
    certificates: CertificateListResponse;
    websites: WebsiteListResponse;
};

type Certificate = CertificateListResponse["items"][number];

const Certificates = ({ certificates, websites }: CertificatesProps) => {
    const { dt } = useTimezone();
    const table = useTable(certificates.pagination);

    const { data, isLoading } = useSWR<CertificateListResponse>(
        `certificates?kind=website&${table.query.slice(1)}`,
        { fallbackData: table.isInitialQuery ? certificates : undefined },
    );

    const columns: TableColumn<Certificate>[] = [
        {
            id: "certificate",
            label: "Certificate",
            isRowHeader: true,
            render: (certificate) => (
                <div className="flex min-w-0 items-center gap-4">
                    <div className="icon-box">
                        <LockKeyhole className="size-5" aria-hidden="true" />
                    </div>
                    <div className="min-w-0">
                        <div className="truncate font-medium">{certificate.name}</div>
                        <div className="mt-1 max-w-xl truncate text-xs text-foreground-400">
                            {certificate.names.length > 1
                                ? `SANs: ${certificate.names.join(", ")}`
                                : certificate.email}
                        </div>
                        {certificate.error && (
                            <div className="mt-1 text-xs text-danger">{certificate.error}</div>
                        )}
                    </div>
                </div>
            ),
        },
        {
            id: "expires",
            label: "Expires",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (certificate) => (certificate.expiresAt ? dt(certificate.expiresAt) : "Never"),
        },
        {
            id: "status",
            label: "Status",
            render: (certificate) => (
                <div className="flex items-center gap-2">
                    <span
                        className={cn(
                            "rounded-full px-2.5 py-1 text-xs capitalize",
                            statusClass(certificate.status),
                        )}
                    >
                        {certificate.status}
                    </span>
                    <span className="text-xs capitalize text-foreground-400">
                        {certificate.kind}
                    </span>
                </div>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (certificate) => <CertificateActions certificate={certificate} />,
        },
    ];

    return (
        <section className="panel-card overflow-hidden">
            <div className="flex items-start justify-between gap-4 px-6 py-5">
                <div>
                    <h2 className="text-base font-semibold">TLS certificates</h2>
                    <div className="mt-1 text-sm text-foreground-500">
                        Let&apos;s Encrypt certificates, expiry and HTTPS redirect policy.
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <CreateCertificate websites={websites} />
                </div>
            </div>
            <Table table={table} columns={columns} data={data} pending={isLoading} />
        </section>
    );
};

export default Certificates;
