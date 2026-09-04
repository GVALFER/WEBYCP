import type { AccountListResponse, NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPage, type PageProps } from "@/utils/paginationServer";
import Accounts from "./accounts";

const AccountsPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/accounts", searchParams);

    const [accounts, nodes] = await Promise.all([
        api.get("accounts", { searchParams: query }).json<AccountListResponse>(),
        api.get("nodes").json<NodeListResponse>(),
    ]);

    await syncPage("/accounts", searchParams, query, accounts.pagination);

    return <Accounts accounts={accounts} nodes={nodes} />;
};

export default AccountsPage;
