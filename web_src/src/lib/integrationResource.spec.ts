import { describe, expect, it } from "vitest";
import { isIntegrationResourceRef, resourceRefLabel } from "@/lib/integrationResource";

describe("integrationResource", () => {
  describe("resourceRefLabel", () => {
    it("returns a plain string value unchanged", () => {
      expect(resourceRefLabel("claude-opus-4-6")).toBe("claude-opus-4-6");
    });

    it("prefers the name of an integration-resource object", () => {
      expect(resourceRefLabel({ id: "abc123", name: "claude-opus-4-6", type: "model" })).toBe("claude-opus-4-6");
    });

    it("falls back to the id when the object has no name", () => {
      expect(resourceRefLabel({ id: "abc123", type: "model" })).toBe("abc123");
    });

    it("stringifies numbers and booleans", () => {
      expect(resourceRefLabel(42)).toBe("42");
      expect(resourceRefLabel(true)).toBe("true");
    });

    it("returns undefined for empty or missing values", () => {
      expect(resourceRefLabel(undefined)).toBeUndefined();
      expect(resourceRefLabel(null)).toBeUndefined();
      expect(resourceRefLabel("")).toBeUndefined();
      expect(resourceRefLabel({})).toBeUndefined();
      expect(resourceRefLabel({ id: "", name: "" })).toBeUndefined();
    });
  });

  describe("isIntegrationResourceRef", () => {
    it("detects resource-ref shaped objects", () => {
      expect(isIntegrationResourceRef({ id: "1", name: "n", type: "model" })).toBe(true);
      expect(isIntegrationResourceRef({ name: "n" })).toBe(true);
    });

    it("rejects scalars, arrays, and unrelated objects", () => {
      expect(isIntegrationResourceRef("model")).toBe(false);
      expect(isIntegrationResourceRef(null)).toBe(false);
      expect(isIntegrationResourceRef([{ id: "1" }])).toBe(false);
      expect(isIntegrationResourceRef({ label: "x" })).toBe(false);
    });
  });
});
