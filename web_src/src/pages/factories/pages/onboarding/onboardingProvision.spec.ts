import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  DEFAULT_LINE_NAME,
  GITHUB_INTAKE_SOURCE,
  provisionEventApps,
  provisionGithubIntake,
  provisionLine,
  provisionPRFeedbackHandler,
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
      defaultBranch: "main",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(result).toEqual({ lineId: "line-1", primaryAppId: "app-1" });
    expect(createLine).not.toHaveBeenCalled();
    expect(installFactory).not.toHaveBeenCalled();
  });

  it("installs plan and implement, and creates a line that runs both", async () => {
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
      defaultBranch: "master",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual([
      "line-planning",
      "line-implementation",
    ]);
    expect(installFactory.mock.calls.map(([input]) => input.installParams)).toEqual([
      { appRepository: "acme/app", backlogRepository: "acme/backlog", defaultBranch: "master" },
      { appRepository: "acme/app", backlogRepository: "acme/backlog", defaultBranch: "master" },
    ]);
    expect(createLine).toHaveBeenCalledWith({
      name: DEFAULT_LINE_NAME,
      steps: [
        {
          type: "runApp",
          app: { app: "canvas-line-planning", entrypoint: "onrun-create-plan" },
        },
        {
          type: "runApp",
          app: { app: "canvas-line-implementation", entrypoint: "onrun-implement" },
        },
      ],
    });
    expect(result.lineId).toBe("line-new");
    expect(updateOnboarding).toHaveBeenCalledWith({
      provisionedAppId: result.primaryAppId,
      provisionedLineId: "line-new",
    });
  });
});

describe("provisionEventApps", () => {
  it("installs PR closure and Create with an Agent for the workspace", async () => {
    const installFactory = vi.fn().mockImplementation(async ({ factoryId }: { factoryId: string }) => ({
      canvasId: `canvas-${factoryId}`,
      canvasName: factoryId,
    }));

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      defaultBranch: "staging",
      installFactory,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["pr-closure", "create-with-agent"]);
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "pr-closure",
        workspaceFactoryId: "factory-1",
        installParams: {
          appRepository: "acme/app",
          backlogRepository: "acme/backlog",
          defaultBranch: "staging",
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

describe("provisionPRFeedbackHandler", () => {
  it("creates a handler for a workspace that has none", async () => {
    const listHandlers = vi.fn().mockResolvedValue([]);
    const createHandler = vi.fn().mockResolvedValue({ id: "handler-1" } as FactoriesFactoryPrFeedbackHandler);

    const handler = await provisionPRFeedbackHandler({
      listHandlers,
      createHandler,
      repository: "acme/app",
    });

    expect(createHandler).toHaveBeenCalledWith({ repository: "acme/app" });
    expect(handler.id).toBe("handler-1");
  });

  it("leaves a handler for the same repository alone so a retry adds no second copy", async () => {
    const listHandlers = vi
      .fn()
      .mockResolvedValue([{ id: "handler-1", settings: { subject: { repository: "acme/app" } } }]);
    const createHandler = vi.fn();

    const handler = await provisionPRFeedbackHandler({
      listHandlers,
      createHandler,
      repository: "acme/app",
    });

    expect(createHandler).not.toHaveBeenCalled();
    expect(handler.id).toBe("handler-1");
  });

  it("creates a handler next to one that watches a different repository", async () => {
    const listHandlers = vi
      .fn()
      .mockResolvedValue([{ id: "handler-1", settings: { subject: { repository: "acme/other" } } }]);
    const createHandler = vi.fn().mockResolvedValue({ id: "handler-2" } as FactoriesFactoryPrFeedbackHandler);

    const handler = await provisionPRFeedbackHandler({
      listHandlers,
      createHandler,
      repository: "acme/app",
    });

    expect(createHandler).toHaveBeenCalledWith({ repository: "acme/app" });
    expect(handler.id).toBe("handler-2");
  });
});
