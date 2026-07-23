import type { InstallParam } from "@/pages/install/types";

export interface FactoryStartingTask {
  id: string;
  label: string;
  prompt: string;
}

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
  run: FactoryRunDefinition;
  source: { type: "bundled" } | { type: "github"; repo: string };
  installParams: InstallParam[];
  canvasYaml: string;
  consoleYaml: string;
}
