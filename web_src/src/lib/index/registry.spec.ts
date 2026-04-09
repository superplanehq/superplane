import type {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsNode,
  IntegrationsIntegrationDefinition,
  TriggersTrigger,
  WidgetsWidget,
} from "@/api-client";
import { describe, expect, it } from "vitest";
import { Registry } from "./registry";

function buildRegistry(overrides?: {
  triggers?: TriggersTrigger[];
  components?: ComponentsComponent[];
  blueprints?: BlueprintsBlueprint[];
  widgets?: WidgetsWidget[];
  availableIntegrations?: IntegrationsIntegrationDefinition[];
}) {
  return new Registry({
    triggers: overrides?.triggers || [],
    components: overrides?.components || [],
    blueprints: overrides?.blueprints || [],
    widgets: overrides?.widgets || [],
    availableIntegrations: overrides?.availableIntegrations || [],
  });
}

describe("Registry", () => {
  it("merges integration triggers/components into allTriggers/allComponents", () => {
    const registry = buildRegistry({
      triggers: [{ name: "start", label: "Start" } as TriggersTrigger],
      components: [{ name: "if", label: "If" } as ComponentsComponent],
      availableIntegrations: [
        {
          name: "github",
          label: "GitHub",
          triggers: [{ name: "github.push", label: "On Push" } as TriggersTrigger],
          components: [{ name: "github.runWorkflow", label: "Run Workflow" } as ComponentsComponent],
        } as IntegrationsIntegrationDefinition,
      ],
    });

    expect(registry.allTriggers.map((trigger) => trigger.name)).toEqual(["start", "github.push"]);
    expect(registry.allComponents.map((component) => component.name)).toEqual(["if", "github.runWorkflow"]);
  });

  it("resolves metadata lookups by id/name", () => {
    const blueprint = { id: "bp-1", name: "Deploy Bundle" } as BlueprintsBlueprint;

    const registry = buildRegistry({
      triggers: [{ name: "start", label: "Start" } as TriggersTrigger],
      components: [{ name: "if", label: "If" } as ComponentsComponent],
      blueprints: [blueprint],
      widgets: [{ name: "annotation", label: "Annotation" } as WidgetsWidget],
      availableIntegrations: [{ name: "github", label: "GitHub" } as IntegrationsIntegrationDefinition],
    });

    expect(registry.getTrigger("start")?.label).toBe("Start");
    expect(registry.getComponent("if")?.label).toBe("If");
    expect(registry.getBlueprint("bp-1")?.name).toBe("Deploy Bundle");
    expect(registry.getWidget("annotation")?.label).toBe("Annotation");
    expect(registry.getAvailableIntegrationLabel("github")).toBe("GitHub");
  });

  it("builds default node base names", () => {
    const registry = buildRegistry({
      blueprints: [{ id: "bp-1", name: "Daily Report" } as BlueprintsBlueprint],
    });

    const blueprintNode = {
      type: "TYPE_BLUEPRINT",
      blueprint: { id: "bp-1" },
    } as ComponentsNode;

    const triggerNode = {
      type: "TYPE_TRIGGER",
      trigger: { name: "webhook" },
    } as ComponentsNode;

    const unnamedNode = {
      type: "TYPE_COMPONENT",
    } as ComponentsNode;

    expect(registry.getDefaultNodeBaseName(blueprintNode)).toBe("Daily Report");
    expect(registry.getDefaultNodeBaseName(triggerNode)).toBe("webhook");
    expect(registry.getDefaultNodeBaseName(unnamedNode)).toBe("node");
  });

  it("builds an icon map from core components and triggers", () => {
    const registry = buildRegistry({
      components: [
        { name: "if", icon: "git-branch" } as ComponentsComponent,
        { name: "wait", icon: undefined } as ComponentsComponent,
      ],
      triggers: [
        { name: "schedule", icon: "clock" } as TriggersTrigger,
        { name: "webhook", icon: undefined } as TriggersTrigger,
      ],
      availableIntegrations: [
        {
          name: "github",
          triggers: [{ name: "github.push", icon: "github" } as TriggersTrigger],
          components: [{ name: "github.runWorkflow", icon: "github" } as ComponentsComponent],
        } as IntegrationsIntegrationDefinition,
      ],
    });

    expect(registry.getIconMap()).toEqual({
      if: "git-branch",
      schedule: "clock",
    });
  });
});
