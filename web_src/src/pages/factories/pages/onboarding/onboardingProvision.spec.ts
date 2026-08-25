import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryIntake } from "@/api-client";

import {
  DEFAULT_LINE_NAME,
  GITHUB_INTAKE_SOURCE,
  provisionEventApps,
  provisionGithubIntake,
  provisionLine,
} from "./onboardingProvision";

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

describe("provisionEventApps", () => {
  it("installs PR closure for the workspace", async () => {
    const installFactory = vi.fn().mockImplementation(async ({ factoryId }: { factoryId: string }) => ({
      canvasId: `canvas-${factoryId}`,
      canvasName: factoryId,
    }));

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      installFactory,
    });

    expect(installFactory).toHaveBeenCalledTimes(1);
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "pr-closure",
        workspaceFactoryId: "factory-1",
        installParams: {
          appRepository: "acme/app",
          backlogRepository: "acme/backlog",
        },
      }),
    );
  });
});

describe("provisionGithubIntake", () => {
  it("creates the GitHub intake for a workspace that has none", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" } as FactoriesFactoryIntake);

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
    expect(intake.id).toBe("intake-1");
  });

  it("leaves an existing GitHub intake alone so a retry adds no second copy", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-1", source: GITHUB_INTAKE_SOURCE }]);
    const createIntake = vi.fn();

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).not.toHaveBeenCalled();
    expect(intake.id).toBe("intake-1");
  });

  it("creates the GitHub intake next to an intake of another source", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-1", source: "SOURCE_SENTRY_EXCEPTIONS" }]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-2" } as FactoriesFactoryIntake);

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
    expect(intake.id).toBe("intake-2");
  });
});
