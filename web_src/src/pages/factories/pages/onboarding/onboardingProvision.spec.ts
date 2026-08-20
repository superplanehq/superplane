import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, FactoriesFactory } from "@/api-client";

import { DEFAULT_LINE_NAME, identifyProvisionedEventApp, provisionEventApps, provisionLine } from "./onboardingProvision";

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
  it("installs issue intake and PR closure for the workspace", async () => {
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

    expect(installFactory).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        factoryId: "issue-intake",
        workspaceFactoryId: "factory-1",
        installParams: {
          appRepository: "acme/app",
          backlogRepository: "acme/backlog",
        },
      }),
    );
    expect(installFactory).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        factoryId: "pr-closure",
        workspaceFactoryId: "factory-1",
      }),
    );
  });

  it("skips issue intake when a prior setup attempt already installed it", async () => {
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
      existingAppFactoryIds: ["issue-intake"],
    });

    expect(installFactory).toHaveBeenCalledTimes(1);
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "pr-closure",
        workspaceFactoryId: "factory-1",
      }),
    );
  });

  it("skips PR closure when a prior setup attempt already installed it", async () => {
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
      existingAppFactoryIds: ["pr-closure"],
    });

    expect(installFactory).toHaveBeenCalledTimes(1);
    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "issue-intake",
        workspaceFactoryId: "factory-1",
      }),
    );
  });

  it("reuses event apps loaded from stable factory ids", async () => {
    const installFactory = vi.fn();

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      installFactory,
      loadExistingAppFactoryIds: async () => new Set(["issue-intake", "pr-closure"]),
    });

    expect(installFactory).not.toHaveBeenCalled();
  });
});

describe("identifyProvisionedEventApp", () => {
  it("identifies Issue Intake from its graph", () => {
    const canvas = {
      spec: {
        nodes: [
          { id: "on-issue-labeled", component: "github.onIssue", configuration: { actions: ["labeled"] } },
          { id: "on-issue-assigned", component: "github.onIssue", configuration: { actions: ["assigned"] } },
          { id: "find-existing-work-order", component: "findWorkOrder" },
          { id: "create-work-order", component: "createWorkOrder" },
        ],
      },
    } as CanvasesCanvas;

    expect(identifyProvisionedEventApp(canvas)).toBe("issue-intake");
  });

  it("identifies PR Closure from its graph", () => {
    const canvas = {
      spec: {
        nodes: [
          { id: "on-pr-closed", component: "github.onPullRequest", configuration: { actions: ["closed"] } },
          { id: "find-work-order", component: "findWorkOrder" },
          { id: "stamp-pr-merged", component: "updateWorkOrderArtifact" },
          { id: "stamp-pr-closed", component: "updateWorkOrderArtifact" },
          { id: "complete-work-order", component: "updateWorkOrderStatus" },
          { id: "reject-work-order", component: "updateWorkOrderStatus" },
        ],
      },
    } as CanvasesCanvas;

    expect(identifyProvisionedEventApp(canvas)).toBe("pr-closure");
  });

  it("returns undefined for unrelated apps", () => {
    const canvas = {
      spec: {
        nodes: [{ id: "onrun-create-plan", component: "onRun" }],
      },
    } as CanvasesCanvas;

    expect(identifyProvisionedEventApp(canvas)).toBeUndefined();
  });
});
