import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoryApp } from "@/api-client";

import { FactoryLineStepEditor } from "./FactoryLineStepEditor";
import type { DraftStep } from "./lib/factoryLineFormShared";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isLoading: false }),
}));

const apps: FactoryApp[] = [{ id: "app-1", name: "Refund Implementer" }];
const appById = new Map(apps.map((app) => [app.id!, app]));

describe("FactoryLineStepEditor", () => {
  // Radix's Select emits its controlled/uncontrolled mismatch warning via
  // `console.warn` (see @radix-ui/react-use-controllable-state), which is
  // what regression tests for this bug need to spy on.
  let consoleWarnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleWarnSpy.mockRestore();
  });

  it("keeps the app and trigger Selects controlled as the step's appId/entrypoint fill in", () => {
    const onChange = vi.fn();
    const emptyStep: DraftStep = { appId: "", entrypoint: "", maxParallelism: "" };
    const { rerender } = render(
      <FactoryLineStepEditor
        organizationId="org-1"
        index={0}
        step={emptyStep}
        apps={apps}
        appById={appById}
        onChange={onChange}
      />,
    );

    const filledStep: DraftStep = { appId: "app-1", entrypoint: "start", maxParallelism: "" };
    rerender(
      <FactoryLineStepEditor
        organizationId="org-1"
        index={0}
        step={filledStep}
        apps={apps}
        appById={appById}
        onChange={onChange}
      />,
    );

    // Clearing the appId (e.g. the user removes the selection) should not
    // re-trigger the warning either.
    rerender(
      <FactoryLineStepEditor
        organizationId="org-1"
        index={0}
        step={emptyStep}
        apps={apps}
        appById={appById}
        onChange={onChange}
      />,
    );

    const warnings = consoleWarnSpy.mock.calls
      .map((args) => args.join(" "))
      .filter((message) => message.includes("controlled"));
    expect(warnings).toEqual([]);
  });
});
