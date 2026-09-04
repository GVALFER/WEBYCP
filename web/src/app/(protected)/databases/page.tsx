import type {
    AccountListResponse,
    DatabaseGrantListResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
} from "@/contracts/types";
import { api } from "@/lib/api";
import Databases from "./databases";

const DatabasesPage = async () => {
    const [accounts, databases, users, grants] = await Promise.all([
        api.get("accounts").json<AccountListResponse>(),
        api.get("databases").json<DatabaseListResponse>(),
        api.get("database-users").json<DatabaseUserListResponse>(),
        api.get("database-grants").json<DatabaseGrantListResponse>(),
    ]);

    return <Databases accounts={accounts} databases={databases} users={users} grants={grants} />;
};

export default DatabasesPage;
