import { describe, expect, it } from "vitest";

import { PRIMARY_FACTORY_ID } from "./factoryPageResponses";
import {
  factoryOwnedCanvasFixture,
  REFUND_IMPLEMENTER_APP,
  refundFactoryLineRuns,
  refundLineCanvasFixture,
} from "./factoryOwnedCanvasFixture";

describe("factoryOwnedCanvasFixture", () => {
  it("marks the canvas as owned by the Refunds Factory", () => {
    const fixture = factoryOwnedCanvasFixture(REFUND_IMPLEMENTER_APP);

    expect(fixture.canvasId).toBe(REFUND_IMPLEMENTER_APP.id);
    expect(fixture.canvas?.canvas?.metadata).toMatchObject({
      id: REFUND_IMPLEMENTER_APP.id,
      name: REFUND_IMPLEMENTER_APP.name,
      factoryId: PRIMARY_FACTORY_ID,
    });
  });

  it("includes the canvas runs that Line cards open", () => {
    const fixture = refundLineCanvasFixture();
    const runIds = (refundFactoryLineRuns().runs ?? []).map((run) => run.id);

    expect(fixture.runs?.runs?.map((run) => run.id)).toEqual(runIds);
    expect(runIds).toEqual(["run-implement", "run-implement-2", "run-implement-3", "run-verify-3"]);
  });
});
