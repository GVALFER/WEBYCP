import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { getTimezone, isValidTimezone } from "./timezones";

describe("timezones", () => {
    it("recognizes supported timezones", () => {
        assert.equal(isValidTimezone("Europe/Lisbon"), true);
        assert.equal(isValidTimezone("Europe/Invalid"), false);
        assert.equal(isValidTimezone(undefined), false);
    });

    it("falls back to UTC", () => {
        assert.deepEqual(getTimezone("Europe/Invalid"), {
            label: "UTC (Coordinated Universal Time)",
            value: "UTC",
        });
    });
});
