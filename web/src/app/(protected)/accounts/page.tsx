import type { AccountListResponse, NodeListResponse, PackageListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import Accounts from "./accounts";

const AccountsPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/accounts", searchParams);

    const [accounts, nodes, packages] = await Promise.all([
        api.get("accounts", { searchParams: query }).json<AccountListResponse>(),
        api.get("nodes").json<NodeListResponse>(),
        api.get("packages", { searchParams: { page: 1, size: 100 } }).json<PackageListResponse>(),
    ]);

    await syncPage("/accounts", searchParams, query, accounts.pagination);

    return <Accounts accounts={accounts} nodes={nodes} packages={packages} />;
};

export default AccountsPage;
