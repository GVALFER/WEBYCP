"use client";

import { CornerDownRight, Globe2 } from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { Table, type TableColumn } from "@/components/table/table";
import { useTable } from "@/components/table/useTable";
import type {
    AccountListResponse,
    DomainAliasListResponse,
    DomainListResponse,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { pageKey } from "@/utils/pagination";
import { statusClass } from "@/utils/status";
import AliasActions from "./actions/aliasActions";
import CreateAlias from "./actions/createAlias";
import CreateDomain from "./actions/createDomain";
import DomainActions from "./actions/domainActions";

type DomainsProps = {
    accounts: AccountListResponse;
    domains: DomainListResponse;
    domainOptions: DomainListResponse;
    aliases: DomainAliasListResponse;
    aliasDomainId: string;
};

type Domain = DomainListResponse["items"][number];
type Alias = DomainAliasListResponse["items"][number];

const Domains = ({
    accounts,
    domains: initialDomains,
    domainOptions,
    aliases: initialAliases,
    aliasDomainId,
}: DomainsProps) => {
    const { dt } = useTimezone();
    const [domainId, setDomainId] = useState("");
    const domainsTable = useTable(initialDomains.pagination, "domains");
    const aliasesTable = useTable(initialAliases.pagination, "aliases");

    const { data: accountsData } = useSWR<AccountListResponse>(
        pageKey("accounts", { page: 1, size: 100 }),
        { fallbackData: accounts },
    );
    const { data: domains } = useSWR<DomainListResponse>(
        `domains${domainsTable.query}`,
        { fallbackData: domainsTable.isInitialQuery ? initialDomains : undefined },
    );
    const { data: allDomains } = useSWR<DomainListResponse>(
        pageKey("domains", { page: 1, size: 100 }),
        { fallbackData: domainOptions },
    );

    const activeDomains = allDomains?.items.filter((domain) => domain.status === "active") ?? [];
    const selectedDomain = activeDomains.some((domain) => domain.id === domainId)
        ? domainId
        : activeDomains[0]?.id || "";
    const aliasKey = selectedDomain
        ? `domains/${encodeURIComponent(selectedDomain)}/aliases${aliasesTable.query}`
        : null;

    const { data: aliases } = useSWR<DomainAliasListResponse>(aliasKey, {
        fallbackData:
            aliasesTable.isInitialQuery && selectedDomain === aliasDomainId
                ? initialAliases
                : undefined,
    });

    const accountNames = new Map(accountsData?.items.map((account) => [account.id, account.name]));
    const currentDomain = allDomains?.items.find((domain) => domain.id === selectedDomain);

    const domainColumns: TableColumn<Domain>[] = [
        {
            id: "domain",
            label: "Domain",
            isRowHeader: true,
            render: (domain) => (
                <div className="flex items-center gap-4">
                    <div className="icon-box">
                        <Globe2 className="size-5" aria-hidden="true" />
                    </div>
                    <div>
                        <div className="font-medium">{domain.name}</div>
                        <div className="mt-1 text-xs text-foreground-400">
                            {accountNames.get(domain.accountId) ?? domain.accountId} · PHP{" "}
                            {domain.phpVersion}
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (domain) => dt(domain.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (domain) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(domain.status),
                    )}
                >
                    {domain.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (domain) => <DomainActions domain={domain} />,
        },
    ];

    const aliasColumns: TableColumn<Alias>[] = [
        {
            id: "alias",
            label: "Alias",
            isRowHeader: true,
            render: (alias) => (
                <div className="flex items-center gap-3">
                    <CornerDownRight
                        className="size-4 text-foreground-400"
                        aria-hidden="true"
                    />
                    <div className="font-medium">{alias.name}</div>
                </div>
            ),
        },
        {
            id: "created",
            label: "Created",
            cellClassName: "whitespace-nowrap text-foreground-500",
            render: (alias) => dt(alias.createdAt),
        },
        {
            id: "status",
            label: "Status",
            render: (alias) => (
                <span
                    className={cn(
                        "rounded-full px-2.5 py-1 text-xs capitalize",
                        statusClass(alias.status),
                    )}
                >
                    {alias.status}
                </span>
            ),
        },
        {
            id: "actions",
            label: "Actions",
            headerClassName: "text-end",
            cellClassName: "w-px whitespace-nowrap",
            render: (alias) => <AliasActions alias={alias} domainId={selectedDomain} />,
        },
    ];

    const setAliasDomain = (id: string) => {
        setDomainId(id);
        aliasesTable.setPage(1);
    };

    return (
        <div className="space-y-6">
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Domains</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Nginx sites and isolated document roots.
                        </div>
                    </div>
                    <CreateDomain accounts={accounts} />
                </div>
                <Table table={domainsTable} columns={domainColumns} data={domains} />
            </section>

            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Aliases</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            {currentDomain
                                ? `Names serving the ${currentDomain.name} document root.`
                                : "Select an active domain to manage its aliases."}
                        </div>
                    </div>
                    <CreateAlias
                        domains={allDomains ?? domainOptions}
                        domainId={selectedDomain}
                        onDomainChange={setAliasDomain}
                    />
                </div>
                <Table
                    table={aliasesTable}
                    columns={aliasColumns}
                    data={selectedDomain ? aliases : initialAliases}
                />
            </section>
        </div>
    );
};

export default Domains;
