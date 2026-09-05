import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { swrConfig } from "./swr";

describe("swrConfig", () => {
    it("uses SSR on mount and revalidates changed keys and focus without polling", () => {
        assert.equal(typeof swrConfig.fetcher, "function");
        assert.equal(swrConfig.revalidateOnFocus, true);
        assert.equal(swrConfig.revalidateOnReconnect, false);
        assert.equal(swrConfig.revalidateIfStale, true);
        assert.equal(swrConfig.revalidateOnMount, false);
        assert.equal(swrConfig.keepPreviousData, true);
        assert.equal(swrConfig.refreshInterval, undefined);
    });
});
