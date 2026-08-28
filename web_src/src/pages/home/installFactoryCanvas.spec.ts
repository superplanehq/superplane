import type { FactoryApp } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { getFactoryDefinition } from "./factories";
import { ensureFactoryCanvas, type CreateFactoryCanvasFn } from "./installFactoryCanvas";

const ORGANIZATION_ID = "org-1";
const WORKSPACE_ID = "factory-1";

function createCanvasSpy(): CreateFactoryCanvasFn & ReturnType<typeof vi.fn> {
  return vi.fn(async (input: { name: string }) => ({
    data: { canvas: { metadata: { id: "canvas-1", name: input.name } } },
  }));
}

function seedQueryClient(args: { workspaceApps?: FactoryApp[]; organizationApps?: { name: string }[] }) {
  const queryClient = new QueryClient();
  queryClient.setQueryData(factoryAppsKey(ORGANIZATION_ID, WORKSPACE_ID), args.workspaceApps ?? []);
  queryClient.setQueryData(canvasKeys.list(ORGANIZATION_ID), args.organizationApps ?? []);
  return queryClient;
}

async function install(args: {
  queryClient: QueryClient;
  workspaceFactoryId?: string;
  createCanvas: CreateFactoryCanvasFn;
}) {
  return ensureFactoryCanvas({
    pending: null,
    organizationId: ORGANIZATION_ID,
    queryClient: args.queryClient,
    definition: getFactoryDefinition("line-planning"),
    workspaceFactoryId: args.workspaceFactoryId,
    createCanvas: args.createCanvas,
    updateCanvasFolderMembership: vi.fn(),
  });
}

describe("ensureFactoryCanvas", () => {
  it("keeps the template name when only another scope holds it", async () => {
    const createCanvas = createCanvasSpy();

    const created = await install({
      queryClient: seedQueryClient({ organizationApps: [{ name: "Plan" }] }),
      workspaceFactoryId: WORKSPACE_ID,
      createCanvas,
    });

    expect(created.canvasName).toBe("Plan");
    expect(createCanvas).toHaveBeenCalledWith(expect.objectContaining({ name: "Plan" }));
  });

  it("steps aside when the same workspace already holds the name", async () => {
    const createCanvas = createCanvasSpy();

    const created = await install({
      queryClient: seedQueryClient({ workspaceApps: [{ name: "Plan" }] }),
      workspaceFactoryId: WORKSPACE_ID,
      createCanvas,
    });

    expect(created.canvasName).toBe("Plan (2)");
  });

  it("steps aside for organization apps when no workspace owns the canvas", async () => {
    const createCanvas = createCanvasSpy();

    const created = await install({
      queryClient: seedQueryClient({ organizationApps: [{ name: "Plan" }] }),
      createCanvas,
    });

    expect(created.canvasName).toBe("Plan (2)");
  });
});
