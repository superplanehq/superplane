import type { FactoryApp } from "@/api-client";
import { defaultCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";

import { FACTORIES_ORGANIZATION_ID, type FactoriesFixture } from "./factoryPageResponses";

type CanvasBody = { metadata?: Record<string, unknown>; spec?: unknown };

const CANVASES_API_ID = /^\/api\/v1\/canvases\/([^/]+)/;

export function canvasIdFromCanvasesApiPath(pathname: string): string | undefined {
  return CANVASES_API_ID.exec(pathname)?.[1];
}

export function findFactoryOwnedApp(
  factoriesFixture: FactoriesFixture,
  canvasId: string,
): { app: FactoryApp; factoryId: string } | undefined {
  for (const [factoryId, apps] of Object.entries(factoriesFixture.appsByFactoryId)) {
    const app = apps.find((entry) => entry.id === canvasId);
    if (app) {
      return { app, factoryId };
    }
  }
  return undefined;
}

/**
 * Software Factory node spec stamped as a factory-owned canvas so Storybook
 * FactoryAppCanvasPage keeps the canvas instead of redirecting to Overview.
 */
export function factoryOwnedCanvasFixture(
  app: FactoryApp,
  factoryId: string,
  overrides: Partial<CanvasAppFixture> = {},
): CanvasAppFixture {
  const canvasId = app.id ?? "";
  const baseCanvas = defaultCanvasAppFixture.canvas?.canvas as CanvasBody | undefined;
  const metadata = {
    ...(baseCanvas?.metadata ?? {}),
    id: canvasId,
    name: app.name,
    description: app.description,
    factoryId,
  };

  return {
    ...defaultCanvasAppFixture,
    organizationId: FACTORIES_ORGANIZATION_ID,
    canvasId,
    canvas: {
      canvas: {
        ...baseCanvas,
        metadata,
      },
    },
    ...overrides,
  };
}

export function resolveFactoryCanvasAppFixture(
  pathname: string,
  appFixture: CanvasAppFixture | undefined,
  factoriesFixture: FactoriesFixture | undefined,
): CanvasAppFixture | undefined {
  if (appFixture) {
    return appFixture;
  }
  if (!factoriesFixture) {
    return undefined;
  }
  const canvasId = canvasIdFromCanvasesApiPath(pathname);
  if (!canvasId) {
    return undefined;
  }
  const owned = findFactoryOwnedApp(factoriesFixture, canvasId);
  if (!owned) {
    return undefined;
  }
  return factoryOwnedCanvasFixture(owned.app, owned.factoryId);
}
