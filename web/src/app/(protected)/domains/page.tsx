import type {
    AccountListResponse,
    DomainAliasListResponse,
    DomainListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPages, type PageProps } from "@/utils/paginationServer";
import Domains from "./domains";

const DomainsPage = async ({ searchParams }: PageProps) => {
    const [domainsQuery, aliasesQuery] = await Promise.all([
        getPageQuery("/domains", searchParams, "domains"),
        getPageQuery("/domains", searchParams, "aliases"),
    ]);

    const [accounts, domains, domainOptions] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("domains", { searchParams: domainsQuery }).json<DomainListResponse>(),
        api
            .get("domains", { searchParams: { page: 1, size: 100 } })
            .json<DomainListResponse>(),
    ]);

    const aliasDomainId = domainOptions.items.find((domain) => domain.status === "active")?.id ?? "";

    const aliases = aliasDomainId
        ? await api
              .get(`domains/${encodeURIComponent(aliasDomainId)}/aliases`, {
                  searchParams: aliasesQuery,
              })
              .json<DomainAliasListResponse>()
        : {
              items: [],
              pagination: {
                  page: aliasesQuery.page,
                  size: aliasesQuery.size,
                  totalItems: 0,
                  totalPages: 0,
              },
          };

    await syncPages("/domains", searchParams, [
        { name: "domains", query: domainsQuery, pagination: domains.pagination },
        { name: "aliases", query: aliasesQuery, pagination: aliases.pagination },
    ]);

    return (
        <Domains
            accounts={accounts}
            domains={domains}
            domainOptions={domainOptions}
            aliases={aliases}
            aliasDomainId={aliasDomainId}
        />
    );
};

export default DomainsPage;
