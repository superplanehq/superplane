import { factoryMissionsPath } from "../../lib/factoryPagePaths";

/** Storybook-only permalink for one mission. */
export function factoryMissionDetailPath(organizationId: string, factoryKey: string, missionId: string) {
  return `${factoryMissionsPath(organizationId, factoryKey)}/${missionId}`;
}
