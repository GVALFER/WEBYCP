"use client";

import { CornerDownRight, Globe2 } from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import type {
    AccountListResponse,
    DomainAliasListResponse,
    DomainListResponse,
} from "@/contracts/types";
import { useTimezone } from "@/hooks/useDate";
import { cn } from "@/utils/classnames";
import { statusClass } from "@/utils/status";
import AliasActions from "./actions/aliasActions";
import CreateAlias from "./actions/createAlias";
import CreateDomain from "./actions/createDomain";
import DomainActions from "./actions/domainActions";

type DomainsProps = {
    accounts: AccountListResponse;
    domains: DomainListResponse;
    aliases: DomainAliasListResponse;
    aliasDomainId: string;
};

const Domains = ({
    accounts,
    domains: initialDomains,
    aliases: initialAliases,
    aliasDomainId,
}: DomainsProps) => {
    const { dt } = useTimezone();
    const [domainId, setDomainId] = useState("");

    const { data: accountsData } = useSWR<AccountListResponse>("accounts", {
        fallbackData: accounts,
    });
    const { data: domains } = useSWR<DomainListResponse>("domains", {
        fallbackData: initialDomains,
    });

    const activeDomains = domains?.items.filter((domain) => domain.status === "active") ?? [];
    const selectedDomain = activeDomains.some((domain) => domain.id === domainId)
        ? domainId
        : activeDomains[0]?.id || "";

    const aliasKey = selectedDomain
        ? `domains/${encodeURIComponent(selectedDomain)}/aliases`
        : null;

    const { data: aliases } = useSWR<DomainAliasListResponse>(aliasKey, {
        fallbackData: selectedDomain === aliasDomainId ? initialAliases : undefined,
    });

    const accountNames = new Map(accountsData?.items.map((account) => [account.id, account.name]));
    const currentDomain = domains?.items.find((domain) => domain.id === selectedDomain);

    return (
        <div className="space-y-6">
            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Domains</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            Nginx sites and isolated document roots.
                        </div>
                    </div>
                    <CreateDomain accounts={accounts} />
                </div>
                <div className="divide-y divide-divider">
                    {domains?.items.length ? (
                        domains.items.map((domain) => (
                            <div
                                key={domain.id}
                                className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                            >
                                <div className="flex items-center gap-4">
                                    <div className="icon-box">
                                        <Globe2 className="size-5" aria-hidden="true" />
                                    </div>
                                    <div>
                                        <div className="font-medium">{domain.name}</div>
                                        <div className="mt-1 text-xs text-foreground-400">
                                            {accountNames.get(domain.accountId) ??
                                                domain.accountId}{" "}
                                            · PHP {domain.phpVersion}
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-4">
                                    <div className="hidden text-xs text-foreground-400 sm:block">
                                        {dt(domain.createdAt)}
                                    </div>
                                    <span
                                        className={cn(
                                            "rounded-full px-2.5 py-1 text-xs capitalize",
                                            statusClass(domain.status),
                                        )}
                                    >
                                        {domain.status}
                                    </span>
                                    <DomainActions domain={domain} />
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className="px-6 py-12 text-center text-sm text-foreground-400">
                            No domains yet.
                        </div>
                    )}
                </div>
            </section>

            <section className="panel-card overflow-hidden">
                <div className="flex items-start justify-between gap-4 border-b border-divider px-6 py-5">
                    <div>
                        <h2 className="text-base font-semibold">Aliases</h2>
                        <div className="mt-1 text-sm text-foreground-500">
                            {currentDomain
                                ? `Names serving the ${currentDomain.name} document root.`
                                : "Select an active domain to manage its aliases."}
                        </div>
                    </div>
                    <CreateAlias
                        domains={initialDomains}
                        domainId={selectedDomain}
                        onDomainChange={setDomainId}
                    />
                </div>
                <div className="divide-y divide-divider">
                    {aliases?.items.length ? (
                        aliases.items.map((alias) => (
                            <div
                                key={alias.id}
                                className="flex items-center justify-between gap-4 px-6 py-4"
                            >
                                <div className="flex items-center gap-3">
                                    <CornerDownRight
                                        className="size-4 text-foreground-400"
                                        aria-hidden="true"
                                    />
                                    <div className="font-medium">{alias.name}</div>
                                </div>
                                <div className="flex items-center gap-2">
                                    <span
                                        className={cn(
                                            "rounded-full px-2.5 py-1 text-xs capitalize",
                                            statusClass(alias.status),
                                        )}
                                    >
                                        {alias.status}
                                    </span>
                                    <AliasActions alias={alias} domainId={selectedDomain} />
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className="px-6 py-10 text-center text-sm text-foreground-400">
                            {selectedDomain ? "No aliases yet." : "No active domain selected."}
                        </div>
                    )}
                </div>
            </section>
        </div>
    );
};

export default Domains;
