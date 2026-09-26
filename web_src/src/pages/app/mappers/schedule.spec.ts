import { describe, expect, it } from "vitest";
import { scheduleTriggerRenderer } from "./schedule";
import type { TriggerRendererContext } from "./types";

function buildContext(concurrencyPolicy?: "allow" | "skip"): TriggerRendererContext {
  return {
    node: {
      id: "schedule-1",
      name: "Nightly checks",
      componentName: "schedule",
      isCollapsed: false,
      configuration: {
        type: "minutes",
        minutesInterval: 5,
        concurrencyPolicy,
      },
      metadata: {
        nextTrigger: "2026-08-03T09:00:00Z",
      },
    },
    definition: {
      name: "schedule",
      label: "Schedule",
      description: "",
      icon: "alarm-clock",
      color: "yellow",
    },
    lastEvent: undefined,
  };
}

describe("scheduleTriggerRenderer.getTriggerProps", () => {
  it("shows when overlapping scheduled runs are skipped", () => {
    const props = scheduleTriggerRenderer.getTriggerProps(buildContext("skip"));

    expect(props.metadata).toContainEqual({
      icon: "ban",
      label: "Skips overlapping runs",
    });
  });

  it.each([undefined, "allow"] as const)("does not add overlap metadata for %s policy", (policy) => {
    const props = scheduleTriggerRenderer.getTriggerProps(buildContext(policy));

    expect(props.metadata).not.toContainEqual({
      icon: "ban",
      label: "Skips overlapping runs",
    });
  });
});
