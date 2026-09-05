import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { AccountListResponse, FTPAccountListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import FTPAccounts from "./ftpAccounts";

const FTPAccountsPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/ftp-accounts", searchParams);

    const [accounts, ftp] = await Promise.all([
        api.get("accounts", { searchParams: { page: 1, size: 100 } }).json<AccountListResponse>(),
        api.get("ftp-accounts", { searchParams: query }).json<FTPAccountListResponse>(),
    ]);

    await syncPage("/ftp-accounts", searchParams, query, ftp.pagination);

    return <FTPAccounts accounts={accounts} ftp={ftp} />;
};

export default FTPAccountsPage;
