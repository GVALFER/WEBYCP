import "server-only";

import type { Pagination } from "@/contracts/types";
import { redirect } from "next/navigation";
import { parseUrlState } from "urlstate-js/server";
import {
    normalizePage,
    pageDefaults,
    pageNames,
    pagePath,
    type PageQuery,
} from "./pagination";

export type PageProps = {
    searchParams: Promise<Record<string, string | string[] | undefined>>;
};

type PageSync = {
    name?: string;
    pagination: Pagination;
    query: PageQuery;
};

export const getPageQuery = async (
    path: string,
    searchParams: PageProps["searchParams"],
    name = "",
) => {
    const names = pageNames(name);
    const values = await searchParams;
    const state = parseUrlState(values, pageDefaults(name));
    const raw = {
        page: state[names.page],
        size: state[names.size],
    };
    const query = normalizePage(raw);

    if (query.page !== raw.page || query.size !== raw.size) {
        redirect(pagePath(path, query, name, values));
    }

    return query;
};

export const syncPages = async (
    path: string,
    searchParams: PageProps["searchParams"],
    pages: PageSync[],
) => {
    let values = await searchParams;
    let target = path;
    let changed = false;

    pages.forEach(({ name = "", pagination, query }) => {
        if (query.page === pagination.page && query.size === pagination.size) return;

        changed = true;
        const url = new URL(pagePath(path, pagination, name, values), "http://webycp.local");
        values = Object.fromEntries(url.searchParams.entries());
        target = `${url.pathname}${url.search}`;
    });

    if (changed) redirect(target);
};

export const syncPage = async (
    path: string,
    searchParams: PageProps["searchParams"],
    query: PageQuery,
    pagination: Pagination,
) => syncPages(path, searchParams, [{ query, pagination }]);
