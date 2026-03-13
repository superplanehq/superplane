import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createDropletMapper } from "./create_droplet";
import { getDropletMapper } from "./get_droplet";
import { deleteDropletMapper } from "./delete_droplet";
import { manageDropletPowerMapper, MANAGE_DROPLET_POWER_STATE_REGISTRY } from "./manage_droplet_power";
import { createSnapshotMapper } from "./create_snapshot";
import { deleteSnapshotMapper } from "./delete_snapshot";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDroplet: createDropletMapper,
  getDroplet: getDropletMapper,
  deleteDroplet: deleteDropletMapper,
  manageDropletPower: manageDropletPowerMapper,
  createSnapshot: createSnapshotMapper,
  deleteSnapshot: deleteSnapshotMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDroplet: buildActionStateRegistry("created"),
  getDroplet: buildActionStateRegistry("fetched"),
  deleteDroplet: buildActionStateRegistry("deleted"),
  manageDropletPower: MANAGE_DROPLET_POWER_STATE_REGISTRY,
  createSnapshot: buildActionStateRegistry("created"),
  deleteSnapshot: buildActionStateRegistry("deleted"),
};
