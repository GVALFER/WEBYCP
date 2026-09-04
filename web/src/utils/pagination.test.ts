import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
    isPageKey,
    normalizePage,
    pageDefaults,
    pageItems,
    pageKey,
    pageNames,
    pagePath,
    pageSearch,
} from "./pagination";

describe("pagination", () => {
    it("normalizes invalid URL values", () => {
        assert.deepEqual(normalizePage({ page: -2, size: 17 }), { page: 1, size: 10 });
        assert.deepEqual(normalizePage({ page: 3, size: 25 }), { page: 3, size: 25 });
    });

    it("creates canonical paths and request keys", () => {
        assert.equal(pagePath("/accounts", { page: 1, size: 10 }), "/accounts");
        assert.equal(pagePath("/accounts", { page: 2, size: 25 }), "/accounts?page=2&size=25");
        assert.equal(
            pagePath("/backups", { page: 2, size: 25 }, "plans"),
            "/backups?plans.page=2&plans.size=25",
        );
        assert.equal(
            pagePath("/backups", { page: 2, size: 10 }, "plans", {
                "artifacts.page": "3",
            }),
            "/backups?artifacts.page=3&plans.page=2",
        );
        assert.equal(
            pagePath("/backups", { page: 1, size: 10 }, "plans", {
                "plans.page": "9",
                "plans.size": "25",
                "artifacts.page": "3",
            }),
            "/backups?artifacts.page=3",
        );
        assert.equal(pageSearch({ page: 2, size: 25 }), "?page=2&size=25");
        assert.equal(pageKey("accounts", { page: 2, size: 25 }), "accounts?page=2&size=25");
        assert.equal(isPageKey("accounts?page=2&size=25", "accounts"), true);
        assert.equal(isPageKey("domains?page=2&size=25", "accounts"), false);
        assert.equal(isPageKey("jobs?page=1&size=10", "accounts", "jobs"), true);
        assert.deepEqual(pageNames("plans"), {
            page: "plans.page",
            size: "plans.size",
        });
        assert.deepEqual(pageDefaults("plans"), {
            "plans.page": 1,
            "plans.size": 10,
        });
    });

    it("keeps the page range compact", () => {
        assert.deepEqual(pageItems(1, 1), [1]);
        assert.deepEqual(pageItems(1, 8), [1, 2, "ellipsis-2", 8]);
        assert.deepEqual(pageItems(4, 8), [1, "ellipsis-1", 3, 4, 5, "ellipsis-5", 8]);
        assert.deepEqual(pageItems(8, 8), [1, "ellipsis-1", 7, 8]);
    });
});
