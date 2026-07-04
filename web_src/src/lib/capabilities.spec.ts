import type { IntegrationsCapabilityDefinition, IntegrationsIntegrationDefinition } from "@/api-client";
import { describe, expect, it } from "vitest";
import {
  actionsFromCapabilities,
  buildIntegrationCapabilityGroupSections,
  triggersFromCapabilities,
} from "@/lib/capabilities";

function capability(name: string, label: string): IntegrationsCapabilityDefinition {
  return { name, label };
}

describe("buildIntegrationCapabilityGroupSections", () => {
  it("returns one unlabeled section listing all definitions when capability groups are absent", () => {
    const defs = [capability("c", "C label"), capability("a", "A label"), capability("b", "B label")];
    expect(buildIntegrationCapabilityGroupSections(undefined, defs)).toEqual([
      { key: "all", label: "", names: ["a", "b", "c"] },
    ]);
  });

  it("sorts group sections alphabetically by group label", () => {
    const definition: IntegrationsIntegrationDefinition = {
      capabilityGroups: [
        { label: "Triggers", capabilities: ["t1"] },
        { label: "Actions", capabilities: ["a1"] },
      ],
    };
    const defs = [capability("a1", "Action one"), capability("t1", "Trigger one")];
    const sections = buildIntegrationCapabilityGroupSections(definition, defs);
    expect(sections.map((section) => section.label)).toEqual(["Actions", "Triggers"]);
    expect(sections[0]?.names).toEqual(["a1"]);
    expect(sections[1]?.names).toEqual(["t1"]);
  });

  it("sorts the Other bucket with the rest by label", () => {
    const definition: IntegrationsIntegrationDefinition = {
      capabilityGroups: [{ label: "Zebra", capabilities: ["z"] }],
    };
    const defs = [capability("z", "Z"), capability("o", "Orphan")];
    const sections = buildIntegrationCapabilityGroupSections(definition, defs);
    expect(sections.map((section) => section.label)).toEqual(["Other", "Zebra"]);
    expect(sections[0]?.names).toEqual(["o"]);
    expect(sections[1]?.names).toEqual(["z"]);
  });

  it("sorts capability names inside a group by definition label, not by group order", () => {
    const definition: IntegrationsIntegrationDefinition = {
      capabilityGroups: [{ label: "One", capabilities: ["second", "first"] }],
    };
    const defs = [capability("first", "B label"), capability("second", "A label")];
    expect(buildIntegrationCapabilityGroupSections(definition, defs)[0]?.names).toEqual(["second", "first"]);
  });

  it("omits empty groups and moves unmatched definitions under Other", () => {
    const definition: IntegrationsIntegrationDefinition = {
      capabilityGroups: [{ label: "Empty", capabilities: ["missing"] }],
    };
    const defs = [capability("only", "Only def")];
    expect(buildIntegrationCapabilityGroupSections(definition, defs)).toEqual([
      { key: "other", label: "Other", names: ["only"] },
    ]);
  });

  it("uses a sequential fallback label when a group label is blank", () => {
    const definition: IntegrationsIntegrationDefinition = {
      capabilityGroups: [{ capabilities: ["x"] }],
    };
    expect(buildIntegrationCapabilityGroupSections(definition, [capability("x", "X")])[0]?.label).toBe("Group 1");
  });
});

describe("capability conversion", () => {
  it("preserves action example output for expression autocomplete fallback", () => {
    const actions = actionsFromCapabilities([
      {
        type: "TYPE_ACTION",
        name: "custom.action",
        label: "Custom Action",
        exampleOutput: { data: { status: "passed" } },
      },
    ]);

    expect(actions[0]).toMatchObject({
      name: "custom.action",
      exampleOutput: { data: { status: "passed" } },
    });
  });

  it("preserves trigger example data for run title preview fallback", () => {
    const triggers = triggersFromCapabilities([
      {
        type: "TYPE_TRIGGER",
        name: "github.onCheckRun",
        label: "Check Run",
        exampleData: {
          data: {
            check_run: {
              name: "DCO",
            },
          },
        },
      },
    ]);

    expect(triggers[0]).toMatchObject({
      name: "github.onCheckRun",
      exampleData: {
        data: {
          check_run: {
            name: "DCO",
          },
        },
      },
    });
  });
});
