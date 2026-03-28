import type { SuperplaneBlueprintsOutputChannel, SuperplaneComponentsOutputChannel } from "@/api-client";

export interface BuildingBlock {
  name: string;
  label?: string;
  description?: string;
  type: "trigger" | "component" | "blueprint";
  componentSubtype?: "trigger" | "action" | "flow";
  outputChannels?: Array<SuperplaneComponentsOutputChannel | SuperplaneBlueprintsOutputChannel>;
  configuration?: any[];
  icon?: string;
  color?: string;
  id?: string;
  integrationName?: string;
  deprecated?: boolean;
  exampleOutput?: Record<string, unknown>;
  exampleData?: Record<string, unknown>;
}

export type BuildingBlockCategory = {
  name: string;
  blocks: BuildingBlock[];
};
