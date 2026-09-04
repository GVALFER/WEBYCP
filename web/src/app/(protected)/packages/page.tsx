import { getPageQuery, syncPage, type PageProps } from "@/components/table/paginationServer";
import type { PackageListResponse } from "@/contracts/types";
import { api } from "@/lib/api";
import Packages from "./packages";

const PackagesPage = async ({ searchParams }: PageProps) => {
    const query = await getPageQuery("/packages", searchParams);

    const packages = await api.get("packages", { searchParams: query }).json<PackageListResponse>();

    await syncPage("/packages", searchParams, query, packages.pagination);

    return <Packages packages={packages} />;
};

export default PackagesPage;
