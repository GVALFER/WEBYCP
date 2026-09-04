import ft from "reqly-js";

let csrfToken = "";

export const api = ft.create({
  baseUrl: "/",
  prefix: "api/v1",
  headers: { accept: "application/json" },
  timeout: 10_000,
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
