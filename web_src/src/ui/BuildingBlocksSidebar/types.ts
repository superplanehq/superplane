import type { ActionsOutputChannel, ConfigurationField } from "@/api-client";

export interface BuildingBlock {
  name: string;
  label?: string;
  description?: string;
  type: "trigger" | "component";
  outputChannels?: Array<ActionsOutputChannel>;
  configuration?: ConfigurationField[];
  icon?: string;
  color?: string;
  id?: string;
  integrationName?: string;
}

export type BuildingBlockCategory = {
  name: string;
  blocks: BuildingBlock[];
};
