export const PAGE_SIZES = [10, 25, 50, 100] as const;

export const pageQuery = {
    page: 1,
    size: 10,
} as const;

export type PageQuery = {
    page: number;
    size: number;
};

export type SearchValues = Record<string, string | string[] | undefined>;

export type PageItem = number | `ellipsis-${number}`;

export const pageNames = (name = "") => ({
    page: name ? `${name}.page` : "page",
    size: name ? `${name}.size` : "size",
});

export const pageDefaults = (name = "") => {
    const names = pageNames(name);
    return {
        [names.page]: pageQuery.page,
        [names.size]: pageQuery.size,
    };
};

export const normalizePage = (query: PageQuery): PageQuery => ({
    page: Number.isInteger(query.page) && query.page > 0 ? query.page : pageQuery.page,
    size: PAGE_SIZES.includes(query.size as (typeof PAGE_SIZES)[number])
        ? query.size
        : pageQuery.size,
});

const searchValues = (values?: SearchValues) => {
    const params = new URLSearchParams();
    if (!values) return params;

    Object.entries(values).forEach(([key, value]) => {
        if (Array.isArray(value)) {
            value.forEach((item) => params.append(key, item));
        } else if (value !== undefined) {
            params.set(key, value);
        }
    });
    return params;
};

export const pagePath = (
    path: string,
    query: PageQuery,
    name = "",
    values?: SearchValues,
) => {
    const names = pageNames(name);
    const params = searchValues(values);
    if (query.page === pageQuery.page) params.delete(names.page);
    else params.set(names.page, String(query.page));
    if (query.size === pageQuery.size) params.delete(names.size);
    else params.set(names.size, String(query.size));
    const search = params.toString();
    return search ? `${path}?${search}` : path;
};

export const pageSearch = (query: PageQuery) => {
    const params = new URLSearchParams({
        page: String(query.page),
        size: String(query.size),
    });
    return `?${params}`;
};

export const pageKey = (path: string, query: PageQuery) => `${path}${pageSearch(query)}`;

export const isPageKey = (key: unknown, ...paths: string[]) =>
    typeof key === "string" &&
    paths.some((path) => key === path || key.startsWith(`${path}?`));

export const pageItems = (page: number, total: number): PageItem[] => {
    if (total <= 0) return [];

    const visible = [1, page - 1, page, page + 1, total]
        .filter((value) => value >= 1 && value <= total)
        .filter((value, index, values) => values.indexOf(value) === index)
        .sort((a, b) => a - b);

    return visible.flatMap((value, index) => {
        const previous = visible[index - 1];
        return previous && value - previous > 1
            ? [`ellipsis-${previous}` as const, value]
            : [value];
    });
};
