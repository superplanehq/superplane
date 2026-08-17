import { describe, expect, it } from "vitest";

import { defaultCanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { SOFTWARE_FACTORY_APP_ID } from "@/pages/home/__fixtures__/homePageResponses";

import {
  canvasIdFromCanvasesApiPath,
  factoryOwnedCanvasFixture,
  findFactoryOwnedApp,
  resolveFactoryCanvasAppFixture,
} from "./factoryOwnedCanvasFixture";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_APPS,
} from "./factoryPageResponses";

const plannerApp = REFUND_FACTORY_APPS[0];

describe("factoryOwnedCanvasFixture", () => {
  it("stamps factory ownership on the default canvas spec", () => {
    const fixture = factoryOwnedCanvasFixture(plannerApp, PRIMARY_FACTORY_ID);

    expect(fixture.organizationId).toBe(FACTORIES_ORGANIZATION_ID);
    expect(fixture.canvasId).toBe(plannerApp.id);
    expect(fixture.canvas?.canvas?.metadata).toMatchObject({
      id: plannerApp.id,
      name: plannerApp.name,
      description: plannerApp.description,
      factoryId: PRIMARY_FACTORY_ID,
    });
    expect(fixture.canvas?.canvas?.spec).toEqual(defaultCanvasAppFixture.canvas?.canvas?.spec);
  });

  it("keeps explicit overrides such as runs", () => {
    const fixture = factoryOwnedCanvasFixture(plannerApp, PRIMARY_FACTORY_ID, {
      runs: { runs: [{ id: "run-implement" }], totalCount: 1, hasNextPage: false },
    });

    expect(fixture.runs?.runs).toEqual([{ id: "run-implement" }]);
    expect(fixture.canvas?.canvas?.metadata).toMatchObject({ factoryId: PRIMARY_FACTORY_ID });
  });
});

describe("findFactoryOwnedApp", () => {
  it("returns the factory app for a known canvas id", () => {
    expect(findFactoryOwnedApp(defaultFactoriesFixture, "app-refund-planner")).toEqual({
      app: plannerApp,
      factoryId: PRIMARY_FACTORY_ID,
    });
  });

  it("returns undefined for the home Software Factory canvas", () => {
    expect(findFactoryOwnedApp(defaultFactoriesFixture, SOFTWARE_FACTORY_APP_ID)).toBeUndefined();
  });
});

describe("resolveFactoryCanvasAppFixture", () => {
  it("prefers an explicit app fixture", () => {
    const explicit = factoryOwnedCanvasFixture(plannerApp, "other-factory");
    const resolved = resolveFactoryCanvasAppFixture(
      "/api/v1/canvases/app-refund-planner",
      explicit,
      defaultFactoriesFixture,
    );

    expect(resolved).toBe(explicit);
  });

  it("builds a factory-owned fixture when the canvas id is a factory app", () => {
    const resolved = resolveFactoryCanvasAppFixture(
      "/api/v1/canvases/app-refund-planner/runs",
      undefined,
      defaultFactoriesFixture,
    );

    expect(resolved?.canvas?.canvas?.metadata).toMatchObject({
      id: "app-refund-planner",
      factoryId: PRIMARY_FACTORY_ID,
    });
  });

  it("leaves Software Factory canvas ids on the default fixture", () => {
    expect(
      resolveFactoryCanvasAppFixture(`/api/v1/canvases/${SOFTWARE_FACTORY_APP_ID}`, undefined, defaultFactoriesFixture),
    ).toBeUndefined();
  });
});

describe("canvasIdFromCanvasesApiPath", () => {
  it("reads the canvas id from describe and nested canvas routes", () => {
    expect(canvasIdFromCanvasesApiPath("/api/v1/canvases/app-refund-planner")).toBe("app-refund-planner");
    expect(canvasIdFromCanvasesApiPath("/api/v1/canvases/app-refund-planner/runs")).toBe("app-refund-planner");
    expect(canvasIdFromCanvasesApiPath("/api/v1/canvases")).toBeUndefined();
  });
});
