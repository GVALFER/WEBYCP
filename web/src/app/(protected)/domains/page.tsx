import type {
    AccountListResponse,
    DomainAliasListResponse,
    DomainListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import Domains from "./domains";

const DomainsPage = async () => {
    const [accounts, domains] = await Promise.all([
        api.get("accounts").json<AccountListResponse>(),
        api.get("domains").json<DomainListResponse>(),
    ]);

    const aliasDomainId = domains.items.find((domain) => domain.status === "active")?.id ?? "";

    const aliases = aliasDomainId
        ? await api
              .get(`domains/${encodeURIComponent(aliasDomainId)}/aliases`)
              .json<DomainAliasListResponse>()
        : { items: [] };

    return (
        <Domains
            accounts={accounts}
            domains={domains}
            aliases={aliases}
            aliasDomainId={aliasDomainId}
        />
    );
};

export default DomainsPage;
