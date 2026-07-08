import type {
  ActionsAction,
  ConfigurationField,
  SuperplaneComponentsNode as ComponentsNode,
  TriggersTrigger,
} from "@/api-client";

export function buildComponentDefinitionsByName(
  componentDefinitions: ActionsAction[] | undefined,
): Map<string, ActionsAction> {
  return new Map(
    componentDefinitions?.filter((definition) => definition.name).map((definition) => [definition.name!, definition]),
  );
}

export function buildTriggerDefinitionsByName(
  triggerDefinitions: TriggersTrigger[] | undefined,
): Map<string, TriggersTrigger> {
  return new Map(
    triggerDefinitions?.filter((definition) => definition.name).map((definition) => [definition.name!, definition]),
  );
}

export function resolveConfigurationFields({
  workflowNode,
  componentDefinitionsByName,
  triggerDefinitionsByName,
}: {
  workflowNode?: ComponentsNode;
  componentDefinitionsByName: Map<string, ActionsAction>;
  triggerDefinitionsByName: Map<string, TriggersTrigger>;
}): ConfigurationField[] {
  if (!workflowNode?.component) return [];

  if (workflowNode.type === "TYPE_ACTION") {
    return componentDefinitionsByName.get(workflowNode.component)?.configuration ?? [];
  }

  if (workflowNode.type === "TYPE_TRIGGER") {
    return triggerDefinitionsByName.get(workflowNode.component)?.configuration ?? [];
  }

  return [];
}
