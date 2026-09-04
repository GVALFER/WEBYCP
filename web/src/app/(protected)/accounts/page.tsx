import type { AccountListResponse, NodeListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Accounts from "./accounts";

const AccountsPage = async () => {
    const [accounts, nodes] = await Promise.all([
        api.get("accounts").json<AccountListResponse>(),
        api.get("nodes").json<NodeListResponse>(),
    ]);

    return <Accounts accounts={accounts} nodes={nodes} />;
};

export default AccountsPage;
