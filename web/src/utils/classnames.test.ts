import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { cn } from "./classnames";

describe("cn", () => {
    it("joins only enabled class names", () => {
        assert.equal(cn("base", false, undefined, "active", null), "base active");
    });
});
