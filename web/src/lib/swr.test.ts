import { describe, expect, it } from "vitest";

import { swrConfig } from "./swr";

describe("swrConfig", () => {
  it("revalidates on focus without polling", () => {
    expect(swrConfig.revalidateOnFocus).toBe(true);
    expect(swrConfig.revalidateOnReconnect).toBe(false);
    expect(swrConfig.refreshInterval).toBeUndefined();
  });
});
