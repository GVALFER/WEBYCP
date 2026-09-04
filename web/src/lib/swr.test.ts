import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { swrConfig } from "./swr";

describe("swrConfig", () => {
    it("revalidates on focus without polling", () => {
        assert.equal(typeof swrConfig.fetcher, "function");
        assert.equal(swrConfig.revalidateOnFocus, true);
        assert.equal(swrConfig.revalidateOnReconnect, false);
        assert.equal(swrConfig.revalidateIfStale, false);
        assert.equal(swrConfig.revalidateOnMount, undefined);
        assert.equal(swrConfig.refreshInterval, undefined);
    });
});
