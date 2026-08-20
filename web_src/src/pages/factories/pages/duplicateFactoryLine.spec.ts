import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactoryLine, FactoryLineStep } from "@/api-client";

import { duplicateFactoryLine, isFactoryLineNameAlreadyExistsError } from "./duplicateFactoryLine";
import { duplicateLineName } from "./lineCardActions";

const steps: FactoryLineStep[] = [
  { type: "run_app", app: { app: "app-refund-planner", entrypoint: "start-plan" } },
  { type: "run_app", app: { app: "app-refund-verifier", entrypoint: "start-verification" } },
];

function baseLine(overrides: Partial<FactoriesFactoryLine> = {}): FactoriesFactoryLine {
  return { id: "line-1", name: "plan-and-implement", steps, ...overrides };
}

describe("duplicateLineName", () => {
  it("appends copy to the trimmed name", () => {
    expect(duplicateLineName("plan-and-implement")).toBe("plan-and-implement copy");
  });

  it("falls back when the name is empty", () => {
    expect(duplicateLineName("  ")).toBe("Unnamed line copy");
  });
});

describe("isFactoryLineNameAlreadyExistsError", () => {
  it("matches the backend error message", () => {
    expect(isFactoryLineNameAlreadyExistsError(new Error("factory line with the same name already exists"))).toBe(true);
  });

  it("does not match unrelated errors", () => {
    expect(isFactoryLineNameAlreadyExistsError(new Error("network error"))).toBe(false);
  });
});

describe("duplicateFactoryLine", () => {
  it("creates a clone with the source steps and a unique copy name", async () => {
    const createLine = vi.fn().mockResolvedValue({ id: "line-2", name: "plan-and-implement copy", steps });

    const created = await duplicateFactoryLine({ line: baseLine(), createLine });

    expect(createLine).toHaveBeenCalledWith({ name: "plan-and-implement copy", steps });
    expect(created).toEqual({ id: "line-2", name: "plan-and-implement copy", steps });
  });

  it("picks the next unique name when the preferred copy name is already taken", async () => {
    const createLine = vi.fn().mockResolvedValue({ id: "line-3", name: "plan-and-implement copy (2)", steps });

    await duplicateFactoryLine({
      line: baseLine(),
      createLine,
      existingNames: ["plan-and-implement", "plan-and-implement copy"],
    });

    expect(createLine).toHaveBeenCalledWith({ name: "plan-and-implement copy (2)", steps });
  });

  it("retries with the next unique name when the backend reports a name collision", async () => {
    const createLine = vi
      .fn()
      .mockRejectedValueOnce(new Error("factory line with the same name already exists"))
      .mockResolvedValueOnce({ id: "line-4", name: "plan-and-implement copy (2)", steps });

    await duplicateFactoryLine({ line: baseLine(), createLine });

    expect(createLine).toHaveBeenNthCalledWith(1, { name: "plan-and-implement copy", steps });
    expect(createLine).toHaveBeenNthCalledWith(2, { name: "plan-and-implement copy (2)", steps });
  });

  it("does not retry for unrelated errors", async () => {
    const createLine = vi.fn().mockRejectedValue(new Error("network error"));

    await expect(duplicateFactoryLine({ line: baseLine(), createLine })).rejects.toThrow("network error");
    expect(createLine).toHaveBeenCalledTimes(1);
  });

  it("gives up after the max retry attempts and throws", async () => {
    const createLine = vi.fn().mockRejectedValue(new Error("factory line with the same name already exists"));

    await expect(duplicateFactoryLine({ line: baseLine(), createLine })).rejects.toThrow("Failed to create line");
    expect(createLine).toHaveBeenCalledTimes(20);
  });

  it("defaults to an empty steps array when the source line has none", async () => {
    const createLine = vi.fn().mockResolvedValue({ id: "line-5", name: "Unnamed line copy", steps: [] });

    await duplicateFactoryLine({ line: baseLine({ name: undefined, steps: undefined }), createLine });

    expect(createLine).toHaveBeenCalledWith({ name: "Unnamed line copy", steps: [] });
  });
});
