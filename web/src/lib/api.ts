import ft from "reqly-js";
import { handleStatus } from "./status";

let csrfToken = "";

const apiUrl = (process.env.WEBYCP_API_URL || "http://127.0.0.1:8080").replace(/\/+$/, "");

export const api = ft.create({
    baseUrl: {
        client: "/",
        server: apiUrl,
    },
    prefix: "api/v1",
    cache: "no-store",
    credentials: "include",
    timeout: 10_000,
    forwardHeaders: {
        extra: ["cookie"],
    },
    headers: {
        accept: "application/json",
    },
    getHeaders: async () => (await import("next/headers")).headers(),
    onStatus: {
        401: handleStatus,
    },
    beforeRequest: ({ request }) => {
        if (csrfToken && request.method !== "GET" && request.method !== "HEAD") {
            request.headers.set("X-CSRF-Token", csrfToken);
        }
    },
});

export const setCsrfToken = (token = "") => {
    csrfToken = token;
};

export const fetcher = <T>(path: string) => api.get(path).json<T>();
