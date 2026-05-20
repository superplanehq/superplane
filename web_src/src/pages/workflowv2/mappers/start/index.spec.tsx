import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { Trigger } from "@/ui/trigger";
import { startTriggerRenderer } from "./index";
import type { TriggerRendererContext } from "../types";

function makeTriggerContext(overrides?: Partial<TriggerRendererContext>): TriggerRendererContext {
  return {
    node: {
      id: "trigger-1",
      name: "Start",
      componentName: "start",
      isCollapsed: false,
      configuration: {
        templates: [{ name: "Example", payload: { ok: true } }],
      },
      metadata: {},
    },
    definition: {
      name: "start",
      label: "Start",
      description: "",
      icon: "play",
      color: "purple",
    },
    lastEvent: undefined,
    ...overrides,
  };
}

describe("start trigger mapper", () => {
  it("runs with template default payload without opening a modal", async () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const user = userEvent.setup();

    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    await user.click(screen.getByTestId("start-template-run"));

    expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
      template: "Example",
      payload: { ok: true },
    });
    expect(openModal).not.toHaveBeenCalled();
  });

  it("opens the run modal when Edit is clicked", async () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const user = userEvent.setup();

    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext(),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    await user.click(screen.getByTestId("start-template-edit"));

    expect(openModal).toHaveBeenCalledTimes(1);
    expect(invokeNodeTriggerHook).not.toHaveBeenCalled();
  });

  it("uses empty payload when template payload is invalid", async () => {
    const invokeNodeTriggerHook = vi.fn().mockResolvedValue(undefined);
    const openModal = vi.fn();
    const user = userEvent.setup();

    const props = startTriggerRenderer.getTriggerProps({
      ...makeTriggerContext({
        node: {
          id: "trigger-1",
          name: "Start",
          componentName: "start",
          isCollapsed: false,
          configuration: {
            templates: [{ name: "Example", payload: [] as unknown as Record<string, unknown> }],
          },
          metadata: {},
        },
      }),
      canvasMode: "live",
      actions: { invokeNodeTriggerHook, openModal },
    });

    render(<Trigger {...props} canvasMode="live" />);
    await user.click(screen.getByTestId("start-template-run"));

    expect(invokeNodeTriggerHook).toHaveBeenCalledWith("run", {
      template: "Example",
      payload: {},
    });
  });
});
