import { describe, expect, it } from "vitest";
import * as v from "valibot";

import { dbNameField, domainField, emailField } from "./validation";

describe("form validation", () => {
  it("normalizes valid input", () => {
    expect(v.parse(emailField, " admin@example.com ")).toBe(
      "admin@example.com",
    );
    expect(v.parse(domainField, " panel.example.com ")).toBe(
      "panel.example.com",
    );
  });

  it("rejects unsafe database names", () => {
    expect(v.safeParse(dbNameField, "name with spaces").success).toBe(false);
  });
});
