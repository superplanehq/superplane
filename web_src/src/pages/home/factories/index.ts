import type { InstallParam } from "@/pages/install/types";

import type { FactoryDefinition } from "./types";
import factoryMeta from "./software-factory/factory.json";
import factoryParams from "./software-factory/params.json";
import softwareFactoryCanvasYaml from "./software-factory/canvas.yaml?raw";
import softwareFactoryConsoleYaml from "./software-factory/console.yaml?raw";

export type { FactoryDefinition, FactoryStartingTask, FactoryRunDefinition } from "./types";
export {
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  substituteInstallParams,
  wireFactoryIntegrations,
} from "./materializeFactoryTemplate";

function buildSoftwareFactory(): FactoryDefinition {
  return {
    id: factoryMeta.id,
    title: factoryMeta.title,
    description: factoryMeta.description,
    integrations: factoryMeta.integrations,
    componentIntegrations: factoryMeta.componentIntegrations,
    startingTasks: factoryMeta.startingTasks,
    run: factoryMeta.run as FactoryDefinition["run"],
    source: factoryMeta.source as FactoryDefinition["source"],
    installParams: factoryParams.install_params as InstallParam[],
    canvasYaml: softwareFactoryCanvasYaml,
    consoleYaml: softwareFactoryConsoleYaml,
  };
}

const FACTORY_BY_ID: Record<string, FactoryDefinition> = {
  "software-factory": buildSoftwareFactory(),
};

export const DEFAULT_FACTORY_ID = "software-factory";

export function getFactoryDefinition(id: string = DEFAULT_FACTORY_ID): FactoryDefinition {
  const definition = FACTORY_BY_ID[id];
  if (!definition) {
    throw new Error(`Unknown factory definition: ${id}`);
  }
  return definition;
}

export function listFactoryDefinitions(): FactoryDefinition[] {
  return Object.values(FACTORY_BY_ID);
}
