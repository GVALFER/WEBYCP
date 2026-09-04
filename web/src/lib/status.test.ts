import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { getLoginPath, getReturnTo, handleStatus } from "./status";

describe("auth status", () => {
    it("only accepts internal return paths", () => {
        assert.equal(getReturnTo("/domains?page=2"), "/domains?page=2");
        assert.equal(getReturnTo("https://example.com"), "/");
        assert.equal(getReturnTo("//example.com"), "/");
        assert.equal(getReturnTo("/login"), "/");
        assert.equal(getLoginPath("/domains?page=2"), "/login?returnTo=%2Fdomains%3Fpage%3D2");
    });

    it("redirects protected 401 responses but ignores auth requests", async () => {
        let redirectedTo = "";
        const originalWindow = globalThis.window;

        Object.defineProperty(globalThis, "window", {
            configurable: true,
            value: {
                location: {
                    origin: "http://webycp.local",
                    pathname: "/accounts",
                    search: "?page=2",
                    replace: (path: string) => {
                        redirectedTo = path;
                    },
                },
            },
        });

        try {
            await handleStatus({
                attempt: 0,
                isServer: false,
                request: new Request("http://webycp.local/api/v1/auth/login"),
                response: Response.json(
                    { code: "invalid_credentials", message: "Invalid credentials" },
                    { status: 401 },
                ),
            });
            await handleStatus({
                attempt: 0,
                isServer: false,
                request: new Request("http://webycp.local/api/v1/auth/me"),
                response: Response.json(
                    { code: "unauthorized", message: "Authentication is required" },
                    { status: 401 },
                ),
            });
            assert.equal(redirectedTo, "");

            await handleStatus({
                attempt: 0,
                isServer: false,
                request: new Request("http://webycp.local/api/v1/accounts"),
                response: Response.json(
                    { code: "unauthorized", message: "Authentication is required" },
                    { status: 401 },
                ),
            });
            assert.equal(redirectedTo, "/login?returnTo=%2Faccounts%3Fpage%3D2");
        } finally {
            Object.defineProperty(globalThis, "window", {
                configurable: true,
                value: originalWindow,
            });
        }
    });
});
