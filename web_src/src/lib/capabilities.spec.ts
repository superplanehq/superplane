import { describe, expect, it } from "vitest";
import type { IntegrationsCapabilityDefinition } from "@/api-client";
import { triggersFromCapabilities } from "@/lib/capabilities";

describe("triggersFromCapabilities", () => {
  it("preserves default run titles for integration triggers", () => {
    const triggers = triggersFromCapabilities([
      {
        type: "TYPE_TRIGGER",
        name: "github.onPush",
        label: "On push",
        description: "Runs when code is pushed",
        configuration: [],
        defaultRunTitle: "{{ root().data.head_commit.message }}",
      },
    ] as IntegrationsCapabilityDefinition[]);

    expect(triggers).toEqual([
      {
        name: "github.onPush",
        label: "On push",
        description: "Runs when code is pushed",
        configuration: [],
        defaultRunTitle: "{{ root().data.head_commit.message }}",
      },
    ]);
  });
});
