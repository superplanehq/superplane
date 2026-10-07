import type { AgentSuggestion } from "@/ui/CanvasPage";

export interface InstallParam {
  name: string;
  label: string;
  type: string; // "string", "integration-resource", or "secret_picker"
  placeholder?: string;
  description?: string;
  default?: string;
  required: boolean;
  // For type "integration-resource"
  integration?: string; // integration type name (e.g. "digitalocean")
  resourceType?: string; // resource type (e.g. "region", "size", "image")
  useNameAsValue?: boolean; // when true, substitute the resource name instead of the ID
  // For type "secret_picker": prefill this key name in the create-secret dialog
  secretKey?: string;
}

export interface FactoryStartingTask {
  id: string;
  label: string;
  prompt: string;
}

export type { AgentSuggestion };

export interface FactoryRunParameterSource {
  from: "startingTaskPrompt";
}

export interface FactoryRunDefinition {
  nodeId: string;
  hookName: string;
  template: string;
  parameters: Record<string, FactoryRunParameterSource>;
}

export interface FactoryDefinition {
  id: string;
  title: string;
  description: string;
  integrations: string[];
  /** Maps canvas component name → integration type used for wiring. */
  componentIntegrations: Record<string, string>;
  startingTasks: FactoryStartingTask[];
  /** Post-install Agent improvement shortcuts shown on the canvas Agent control. */
  agentSuggestions?: AgentSuggestion[];
  run: FactoryRunDefinition;
  source: { type: "bundled" } | { type: "github"; repo: string };
  installParams: InstallParam[];
  /** Legacy standalone templates. Factory workspace apps are materialized by the backend catalog. */
  canvasYaml?: string;
  consoleYaml?: string;
}
