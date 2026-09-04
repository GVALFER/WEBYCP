import type {
    AccountListResponse,
    DatabaseGrantListResponse,
    DatabaseListResponse,
    DatabaseUserListResponse,
    ServiceSettings,
} from "@/contracts/types";
import { api } from "@/lib/api";
import { getPageQuery, syncPages, type PageProps } from "@/components/table/paginationServer";
import Databases from "./databases";

const DatabasesPage = async ({ searchParams }: PageProps) => {
    const [databasesQuery, usersQuery, grantsQuery] = await Promise.all([
        getPageQuery("/databases", searchParams, "databases"),
        getPageQuery("/databases", searchParams, "users"),
        getPageQuery("/databases", searchParams, "grants"),
    ]);

    const [accounts, databases, users, grants, databaseOptions, userOptions, settings] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("databases", { searchParams: databasesQuery }).json<DatabaseListResponse>(),
        api.get("database-users", { searchParams: usersQuery }).json<DatabaseUserListResponse>(),
        api.get("database-grants", { searchParams: grantsQuery }).json<DatabaseGrantListResponse>(),
        api.get("databases", { searchParams: { page: 1, size: 100 } }).json<DatabaseListResponse>(),
        api
            .get("database-users", { searchParams: { page: 1, size: 100 } })
            .json<DatabaseUserListResponse>(),
        api.get("service-settings").json<ServiceSettings>(),
    ]);

    await syncPages("/databases", searchParams, [
        { name: "databases", query: databasesQuery, pagination: databases.pagination },
        { name: "users", query: usersQuery, pagination: users.pagination },
        { name: "grants", query: grantsQuery, pagination: grants.pagination },
    ]);

    return (
        <Databases
            accounts={accounts}
            databases={databases}
            users={users}
            grants={grants}
            databaseOptions={databaseOptions}
            userOptions={userOptions}
            settings={settings}
        />
    );
};

export default DatabasesPage;
