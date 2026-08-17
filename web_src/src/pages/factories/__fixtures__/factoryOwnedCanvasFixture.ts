import type { FactoryApp } from "@/api-client";
import { defaultCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";

import {
  FACTORIES_ORGANIZATION_ID,
  HOUR_AGO,
  LAST_WEEK,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_APPS,
  TWO_HOURS_AGO,
  YESTERDAY,
} from "./factoryPageResponses";

export const REFUND_PLANNER_APP = REFUND_FACTORY_APPS[0];
export const REFUND_IMPLEMENTER_APP = REFUND_FACTORY_APPS[1];
export const REFUND_VERIFIER_APP = REFUND_FACTORY_APPS[2];

/**
 * Clone of the captured Software Factory canvas, owned by the Refunds Factory
 * so FactoryAppCanvasPage does not redirect to Overview.
 */
export function factoryOwnedCanvasFixture(
  app: Pick<FactoryApp, "id" | "name" | "description"> = REFUND_PLANNER_APP,
  extras: Partial<CanvasAppFixture> = {},
): CanvasAppFixture {
  const baseCanvas = defaultCanvasAppFixture.canvas?.canvas as
    | { metadata?: Record<string, unknown>; spec?: unknown }
    | undefined;
  const canvasId = app.id ?? "app-refund-planner";

  return {
    ...defaultCanvasAppFixture,
    ...extras,
    organizationId: FACTORIES_ORGANIZATION_ID,
    canvasId,
    canvas: extras.canvas ?? {
      canvas: {
        ...baseCanvas,
        metadata: {
          ...(baseCanvas?.metadata ?? {}),
          id: canvasId,
          name: app.name,
          description: app.description,
          factoryId: PRIMARY_FACTORY_ID,
        },
      },
    },
  };
}

/** Canvas runs that Line phase cards link to (`run-implement`, `run-implement-2`, `run-verify-3`). */
export function refundFactoryLineRuns(): NonNullable<CanvasAppFixture["runs"]> {
  const implementerId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
  const verifierId = REFUND_VERIFIER_APP.id ?? "app-refund-verifier";

  return {
    runs: [
      {
        id: "run-implement",
        canvasId: implementerId,
        state: "STATE_STARTED",
        createdAt: HOUR_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: { customName: "Add refund reconciliation test" },
      },
      {
        id: "run-implement-2",
        canvasId: implementerId,
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        createdAt: TWO_HOURS_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: { customName: "Ship idempotent refund retries" },
      },
      {
        id: "run-implement-3",
        canvasId: implementerId,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: LAST_WEEK,
        updatedAt: LAST_WEEK,
        rootEvent: { customName: "Backfill refund audit trail" },
      },
      {
        id: "run-verify-3",
        canvasId: verifierId,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: LAST_WEEK,
        updatedAt: YESTERDAY,
        rootEvent: { customName: "Backfill refund audit trail" },
      },
    ],
    totalCount: 4,
    hasNextPage: false,
  };
}

/** Factory-owned canvas plus the runs that Line cards open. */
export function refundLineCanvasFixture(
  app: Pick<FactoryApp, "id" | "name" | "description"> = REFUND_IMPLEMENTER_APP,
): CanvasAppFixture {
  return factoryOwnedCanvasFixture(app, { runs: refundFactoryLineRuns() });
}
