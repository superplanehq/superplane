import type { FactoryApp } from "@/api-client";
import { defaultCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";

import {
  FACTORIES_ORGANIZATION_ID,
  HOUR_AGO,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_APPS,
  TWO_HOURS_AGO,
  YESTERDAY,
} from "./factoryPageResponses";
import {
  SIMPLE_FACTORY_RUN_EVENT_AT,
  mergeSimpleFactoryRunActions,
  simpleFactoryRunCanvasSpec,
  simpleFactoryRunExecutions,
  simpleFactoryRunOnRunTrigger,
} from "./simpleFactoryRunCanvas";

export {
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
};

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
  const extraCanvas = extras.canvas?.canvas as { metadata?: Record<string, unknown>; spec?: unknown } | undefined;
  const canvasId = app.id ?? "app-refund-planner";
  const { canvas: _canvas, ...fixtureExtras } = extras;

  return {
    ...defaultCanvasAppFixture,
    ...fixtureExtras,
    organizationId: FACTORIES_ORGANIZATION_ID,
    canvasId,
    canvas: {
      canvas: {
        ...baseCanvas,
        ...extraCanvas,
        metadata: {
          ...(baseCanvas?.metadata ?? {}),
          ...(extraCanvas?.metadata ?? {}),
          id: canvasId,
          name: app.name,
          description: app.description,
          factoryId: PRIMARY_FACTORY_ID,
        },
      },
    } as CanvasAppFixture["canvas"],
  };
}

/** Canvas runs that Line phase cards open. Ids are UUIDs so `?run=` stays on AppPage. */
export function refundFactoryLineRuns(): NonNullable<CanvasAppFixture["runs"]> {
  const implementerId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
  const verifierId = REFUND_VERIFIER_APP.id ?? "app-refund-verifier";

  return {
    runs: [
      {
        id: LINE_RUN_IMPLEMENT_ID,
        canvasId: implementerId,
        state: "STATE_STARTED",
        createdAt: HOUR_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: { customName: "Add refund reconciliation test" },
      },
      {
        id: LINE_RUN_IMPLEMENT_FAILED_ID,
        canvasId: implementerId,
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        createdAt: TWO_HOURS_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: {
          id: LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID,
          nodeId: "on-run",
          customName: "Ship idempotent refund retries",
          createdAt: SIMPLE_FACTORY_RUN_EVENT_AT,
        },
      },
      {
        id: LINE_RUN_IMPLEMENT_PASSED_ID,
        canvasId: implementerId,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: LAST_WEEK,
        updatedAt: LAST_WEEK,
        rootEvent: { customName: "Backfill refund audit trail" },
      },
      {
        id: LINE_RUN_VERIFY_PASSED_ID,
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

/** Factory-owned compact CI canvas plus the runs that Line cards open. */
export function refundLineCanvasFixture(
  app: Pick<FactoryApp, "id" | "name" | "description"> = REFUND_IMPLEMENTER_APP,
): CanvasAppFixture {
  return factoryOwnedCanvasFixture(app, {
    runs: refundFactoryLineRuns(),
    triggers: { triggers: [simpleFactoryRunOnRunTrigger()] },
    actions: mergeSimpleFactoryRunActions(defaultCanvasAppFixture.actions),
    executionsByEventId: {
      [LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID]: { executions: simpleFactoryRunExecutions() },
    },
    canvas: {
      canvas: {
        spec: simpleFactoryRunCanvasSpec(),
      },
    },
  });
}
