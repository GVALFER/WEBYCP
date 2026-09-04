import { errorInfo, HTTPError, type OnStatus } from "reqly-js";

const authPaths = new Set(["/api/v1/auth/login", "/api/v1/auth/me"]);
let redirected = false;

export const getReturnTo = (value?: string | null) => {
    if (!value?.startsWith("/")) return "/";

    const base = "http://webycp.local";
    const url = new URL(value, base);

    if (url.origin !== base || url.pathname === "/login") return "/";

    return `${url.pathname}${url.search}${url.hash}`;
};

export const getLoginPath = (returnTo: string) => {
    const path = getReturnTo(returnTo);
    if (path === "/") return "/login";

    return `/login?${new URLSearchParams({ returnTo: path })}`;
};

const isAuthRequest = (url: string) => authPaths.has(new URL(url).pathname);

const redirectTo = async (path: string, isServer: boolean) => {
    if (isServer) {
        const { redirect } = await import("next/navigation");
        redirect(path);
    }

    const target = new URL(path, window.location.origin);
    if (
        redirected ||
        (window.location.pathname === target.pathname && window.location.search === target.search)
    ) {
        return;
    }

    redirected = true;
    window.location.replace(path);
};

export const handleStatus: OnStatus = async ({ isServer, request, response }) => {
    const error = await errorInfo(new HTTPError(response, request));

    if (error.code !== "unauthorized" || isAuthRequest(request.url)) return;

    const returnTo = isServer
        ? "/"
        : getReturnTo(`${window.location.pathname}${window.location.search}`);

    await redirectTo(getLoginPath(returnTo), isServer);
};
