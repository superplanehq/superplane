import { editFactoryLinePath, factoryLineDetailPath } from "./factoryPagePaths";

interface FactoryLineDestination {
  organizationId: string;
  factoryId: string;
  lineId: string;
  canEdit: boolean;
}

export function factoryLineDestinationPath({
  organizationId,
  factoryId,
  lineId,
  canEdit,
}: FactoryLineDestination): string | undefined {
  if (!lineId || lineId === "unknown") {
    return undefined;
  }
  if (canEdit) {
    return editFactoryLinePath(organizationId, factoryId, lineId);
  }
  return factoryLineDetailPath(organizationId, factoryId, lineId);
}
