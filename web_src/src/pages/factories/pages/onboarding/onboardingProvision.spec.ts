import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { DEFAULT_LINE_NAME, provisionLine } from "./onboardingProvision";

describe("provisionLine", () => {
  it("reuses a line that already has the planning entrypoint", async () => {
    const createLine = vi.fn();
    const updateOnboarding = vi.fn();
    const installFactory = vi.fn();
    const factory = {
      id: "factory-1",
      lines: [
        {
          id: "line-1",
          steps: [{ app: { app: "app-1", entrypoint: "onrun-create-plan" } }],
        },
      ],
    } as FactoriesFactory;

    const result = await provisionLine({
      factory,
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(result).toEqual({ lineId: "line-1", primaryAppId: "app-1" });
    expect(createLine).not.toHaveBeenCalled();
    expect(installFactory).not.toHaveBeenCalled();
  });

  it("creates a line named Software delivery when none exists", async () => {
    const createLine = vi.fn().mockResolvedValue({ id: "line-new" });
    const updateOnboarding = vi.fn().mockResolvedValue({});
    const installFactory = vi.fn().mockImplementation(async ({ factoryId }: { factoryId: string }) => ({
      canvasId: `canvas-${factoryId}`,
      canvasName: factoryId,
    }));

    const result = await provisionLine({
      factory: { id: "factory-1" } as FactoriesFactory,
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(createLine).toHaveBeenCalledWith(
      expect.objectContaining({
        name: DEFAULT_LINE_NAME,
      }),
    );
    expect(result.lineId).toBe("line-new");
    expect(updateOnboarding).toHaveBeenCalledWith({
      provisionedAppId: result.primaryAppId,
      provisionedLineId: "line-new",
    });
  });
});
