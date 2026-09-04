import assert from "node:assert/strict";
import { describe, it } from "node:test";
import * as v from "valibot";

import { dbNameField, domainField, emailField, usernameField } from "./validation";

describe("form validation", () => {
    it("normalizes valid input", () => {
        assert.equal(v.parse(emailField, " admin@example.com "), "admin@example.com");
        assert.equal(v.parse(domainField, " panel.example.com "), "panel.example.com");
        assert.equal(v.parse(usernameField, " Owner.Name "), "owner.name");
    });

    it("rejects unsafe database names", () => {
        assert.equal(v.safeParse(dbNameField, "name with spaces").success, false);
    });

    it("rejects invalid usernames", () => {
        assert.equal(v.safeParse(usernameField, "ab").success, false);
        assert.equal(v.safeParse(usernameField, "owner name").success, false);
    });
});
