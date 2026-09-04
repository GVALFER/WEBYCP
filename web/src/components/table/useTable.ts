"use client";

import { useCallback, useEffect, useMemo } from "react";
import { useUrlState } from "urlstate-js";
import {
    normalizePage,
    pageDefaults,
    pageNames,
    pageQuery,
    pageSearch,
    type PageQuery,
} from "@/utils/pagination";

export const useTable = (initial: PageQuery = pageQuery, name = "") => {
    const first = useMemo(
        () => normalizePage({ page: initial.page, size: initial.size }),
        [initial.page, initial.size],
    );

    const names = useMemo(() => pageNames(name), [name]);
    const defaults = useMemo(() => pageDefaults(name), [name]);
    const initialState = useMemo(
        () => ({
            [names.page]: first.page,
            [names.size]: first.size,
        }),
        [first.page, first.size, names.page, names.size],
    );

    const [state, setState] = useUrlState(defaults, { initial: initialState });
    const rawPage = state[names.page];
    const rawSize = state[names.size];

    const page = useMemo(
        () => normalizePage({ page: rawPage, size: rawSize }),
        [rawPage, rawSize],
    );

    useEffect(() => {
        if (page.page !== rawPage || page.size !== rawSize) {
            setState({
                [names.page]: page.page,
                [names.size]: page.size,
            });
        }
    }, [names.page, names.size, page.page, page.size, rawPage, rawSize, setState]);

    const setPage = useCallback(
        (value: number) =>
            setState(
                { [names.page]: Math.max(1, Math.floor(value)) },
                { history: "push" },
            ),
        [names.page, setState],
    );

    const setSize = useCallback(
        (value: number) =>
            setState({
                [names.page]: 1,
                [names.size]: normalizePage({ page: 1, size: value }).size,
            }),
        [names.page, names.size, setState],
    );

    return {
        isInitialQuery: page.page === first.page && page.size === first.size,
        page,
        query: pageSearch(page),
        setPage,
        setSize,
    };
};

export type TableState = ReturnType<typeof useTable>;
