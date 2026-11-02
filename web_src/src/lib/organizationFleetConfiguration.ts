import type { SuperplaneOrganizationsFleet } from "@/api-client";

export type FleetSection = "version" | "platform" | "resources";
export type FleetDraft = {
  fleetId: string;
  runnerVersion: string;
  operatingSystem: string;
  architecture: string;
  cpuMillicores: string;
  memoryMb: string;
  diskGb: string;
  capabilities: string;
};

export function draftFromFleet(fleet: SuperplaneOrganizationsFleet = {}): FleetDraft {
  const spec = fleet.spec ?? {};
  return {
    fleetId: fleet.id ?? "",
    runnerVersion: fleet.runnerVersion ?? "",
    operatingSystem: spec.operatingSystem ?? "linux",
    architecture: spec.architecture ?? "amd64",
    cpuMillicores: String(spec.cpuMillicores ?? 1000),
    memoryMb: String(spec.memoryMb ?? 1024),
    diskGb: String(spec.diskGb ?? 20),
    capabilities: spec.capabilities?.join(", ") ?? "",
  };
}
