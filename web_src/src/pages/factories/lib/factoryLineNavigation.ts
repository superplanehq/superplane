import { editFactoryLinePath, factoryLineDetailPath } from "./factoryPagePaths";

interface FactoryLineDestination {
  organizationId: string;
  factoryKey: string;
  lineId: string;
  canEdit: boolean;
}

export function factoryLineDestinationPath({
  organizationId,
  factoryKey,
  lineId,
  canEdit,
}: FactoryLineDestination): string | undefined {
  if (!lineId || lineId === "unknown") {
    return undefined;
  }
  if (canEdit) {
    return editFactoryLinePath(organizationId, factoryKey, lineId);
  }
  return factoryLineDetailPath(organizationId, factoryKey, lineId);
}
