import { ComponentBaseMapper, EventStateRegistry, TriggerRenderer } from "../types";
import { createDropletMapper } from "./create_droplet";
import { getDropletMapper } from "./get_droplet";
import { deleteDropletMapper } from "./delete_droplet";
import { manageDropletPowerMapper } from "./manage_droplet_power";
import { createDNSRecordMapper } from "./create_dns_record";
import { deleteDNSRecordMapper } from "./delete_dns_record";
import { upsertDNSRecordMapper } from "./upsert_dns_record";
import { buildActionStateRegistry } from "../utils";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createDroplet: createDropletMapper,
  getDroplet: getDropletMapper,
  deleteDroplet: deleteDropletMapper,
  manageDropletPower: manageDropletPowerMapper,
  createDNSRecord: createDNSRecordMapper,
  deleteDNSRecord: deleteDNSRecordMapper,
  upsertDNSRecord: upsertDNSRecordMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createDroplet: buildActionStateRegistry("created"),
  getDroplet: buildActionStateRegistry("fetched"),
  deleteDroplet: buildActionStateRegistry("deleted"),
  manageDropletPower: buildActionStateRegistry("managed"),
  createDNSRecord: buildActionStateRegistry("created"),
  deleteDNSRecord: buildActionStateRegistry("deleted"),
  upsertDNSRecord: buildActionStateRegistry("upserted"),
};
