import type {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsNode,
  IntegrationsIntegrationDefinition,
  TriggersTrigger,
  WidgetsWidget,
} from "@/api-client";
import type { BuildingBlockCategory } from "@/lib/index/types";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";

type RegistryOptions = {
  triggers: TriggersTrigger[];
  components: ComponentsComponent[];
  blueprints: BlueprintsBlueprint[];
  widgets: WidgetsWidget[];
  availableIntegrations: IntegrationsIntegrationDefinition[];
};

function mergeTriggers(
  triggers: TriggersTrigger[],
  availableIntegrations: IntegrationsIntegrationDefinition[],
): TriggersTrigger[] {
  const merged = [...triggers];
  availableIntegrations.forEach((integration) => {
    if (integration.triggers) {
      merged.push(...integration.triggers);
    }
  });
  return merged;
}

function mergeComponents(
  components: ComponentsComponent[],
  availableIntegrations: IntegrationsIntegrationDefinition[],
): ComponentsComponent[] {
  const merged = [...components];
  availableIntegrations.forEach((integration) => {
    if (integration.components) {
      merged.push(...integration.components);
    }
  });
  return merged;
}

function mapByKeyFirstWins<T>(items: T[], getKey: (item: T) => string | undefined): Map<string, T> {
  const lookup = new Map<string, T>();

  items.forEach((item) => {
    const key = getKey(item);
    if (!key || lookup.has(key)) {
      return;
    }

    lookup.set(key, item);
  });

  return lookup;
}

function buildIconMap(components: ComponentsComponent[], triggers: TriggersTrigger[]): Record<string, string> {
  const iconMap: Record<string, string> = {};

  components.forEach((component) => {
    if (component.name && component.icon) {
      iconMap[component.name] = component.icon;
    }
  });

  triggers.forEach((trigger) => {
    if (trigger.name && trigger.icon) {
      iconMap[trigger.name] = trigger.icon;
    }
  });

  return iconMap;
}

export class Registry {
  readonly triggers: TriggersTrigger[];
  readonly components: ComponentsComponent[];
  readonly blueprints: BlueprintsBlueprint[];
  readonly widgets: WidgetsWidget[];
  readonly availableIntegrations: IntegrationsIntegrationDefinition[];

  readonly allTriggers: TriggersTrigger[];
  readonly allComponents: ComponentsComponent[];
  readonly buildingBlocks: BuildingBlockCategory[];

  private readonly triggerByName: Map<string, TriggersTrigger>;
  private readonly componentByName: Map<string, ComponentsComponent>;
  private readonly blueprintById: Map<string, BlueprintsBlueprint>;
  private readonly widgetByName: Map<string, WidgetsWidget>;
  private readonly integrationByName: Map<string, IntegrationsIntegrationDefinition>;
  private readonly iconMap: Record<string, string>;

  constructor(options: RegistryOptions) {
    this.triggers = options.triggers;
    this.components = options.components;
    this.blueprints = options.blueprints;
    this.widgets = options.widgets;
    this.availableIntegrations = options.availableIntegrations;

    this.allTriggers = mergeTriggers(this.triggers, this.availableIntegrations);
    this.allComponents = mergeComponents(this.components, this.availableIntegrations);
    this.buildingBlocks = buildBuildingBlockCategories(
      this.triggers,
      this.components,
      this.availableIntegrations,
      this.widgets,
    );

    this.triggerByName = mapByKeyFirstWins(this.allTriggers, (trigger) => trigger.name || undefined);
    this.componentByName = mapByKeyFirstWins(this.allComponents, (component) => component.name || undefined);
    this.blueprintById = mapByKeyFirstWins(this.blueprints, (blueprint) => blueprint.id || undefined);
    this.widgetByName = mapByKeyFirstWins(this.widgets, (widget) => widget.name || undefined);
    this.integrationByName = mapByKeyFirstWins(
      this.availableIntegrations,
      (integration) => integration.name || undefined,
    );
    this.iconMap = buildIconMap(this.components, this.triggers);
  }

  getTrigger(name?: string | null): TriggersTrigger | undefined {
    if (!name) {
      return undefined;
    }

    return this.triggerByName.get(name);
  }

  getComponent(name?: string | null): ComponentsComponent | undefined {
    if (!name) {
      return undefined;
    }

    return this.componentByName.get(name);
  }

  getBlueprint(id?: string | null): BlueprintsBlueprint | undefined {
    if (!id) {
      return undefined;
    }

    return this.blueprintById.get(id);
  }

  getWidget(name?: string | null): WidgetsWidget | undefined {
    if (!name) {
      return undefined;
    }

    return this.widgetByName.get(name);
  }

  getAvailableIntegration(name?: string | null): IntegrationsIntegrationDefinition | undefined {
    if (!name) {
      return undefined;
    }

    return this.integrationByName.get(name);
  }

  getAvailableIntegrationLabel(name?: string | null): string | undefined {
    return this.getAvailableIntegration(name)?.label || undefined;
  }

  getIconMap(): Record<string, string> {
    return this.iconMap;
  }

  getDefaultNodeBaseName(node: ComponentsNode): string {
    const nodeName = node.name?.trim();
    if (nodeName) {
      return nodeName;
    }

    if (node.type === "TYPE_TRIGGER" && node.trigger?.name) {
      return node.trigger.name;
    }

    if (node.type === "TYPE_COMPONENT" && node.component?.name) {
      return node.component.name;
    }

    if (node.type === "TYPE_BLUEPRINT" && node.blueprint?.id) {
      return this.getBlueprint(node.blueprint.id)?.name || "blueprint";
    }

    return "node";
  }
}
