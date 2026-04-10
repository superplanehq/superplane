import type {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsNodeType,
  TriggersTrigger,
  WidgetsWidget,
} from "@/api-client";

export type BuildingBlockType = ComponentsNodeType;
export type BuildingBlockSubtype = "trigger" | "action" | "flow";

type BuildingBlockMetadata = {
  componentSubtype?: BuildingBlockSubtype;
  integrationName?: string;
  deprecated?: boolean;
};

export type TriggerBuildingBlock = TriggersTrigger &
  BuildingBlockMetadata & {
    type: "TYPE_TRIGGER";
    name: string;
  };

export type ComponentBuildingBlock = ComponentsComponent &
  BuildingBlockMetadata & {
    type: "TYPE_COMPONENT";
    name: string;
  };

export type WidgetBuildingBlock = WidgetsWidget &
  BuildingBlockMetadata & {
    type: "TYPE_WIDGET";
    name: string;
  };

export type BlueprintBuildingBlock = BlueprintsBlueprint &
  BuildingBlockMetadata & {
    type: "TYPE_BLUEPRINT";
    id: string;
    name: string;
  };

export type BuildingBlock =
  | ComponentBuildingBlock
  | TriggerBuildingBlock
  | WidgetBuildingBlock
  | BlueprintBuildingBlock;

export type BuildingBlockCategory = {
  name: string;
  blocks: BuildingBlock[];
};
