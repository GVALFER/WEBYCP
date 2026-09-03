import { describe, expect, it } from "vitest";

import { cn } from "./classnames";

describe("cn", () => {
  it("joins only enabled class names", () => {
    expect(cn("base", false, undefined, "active", null)).toBe("base active");
  });
});
