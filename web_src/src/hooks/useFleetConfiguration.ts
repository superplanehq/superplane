import { useEffect, useRef, useState } from "react";
import {
  organizationsCreateRunnerFleet,
  organizationsUpdateRunnerFleet,
  type SuperplaneOrganizationsFleet,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { draftFromFleet, type FleetDraft, type FleetSection } from "@/lib/organizationFleetConfiguration";

function fleetSpec(draft: FleetDraft) {
  return {
    operatingSystem: draft.operatingSystem.trim(),
    architecture: draft.architecture.trim(),
    cpuMillicores: Number(draft.cpuMillicores),
    memoryMb: Number(draft.memoryMb),
    diskGb: Number(draft.diskGb),
    capabilities: draft.capabilities
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean),
  };
}

function sectionPatch(draft: FleetDraft, fleet: SuperplaneOrganizationsFleet, section: FleetSection) {
  if (section === "version") return { runnerVersion: draft.runnerVersion.trim() };
  const spec = fleetSpec(draft);
  if (section === "platform")
    return { spec: { ...fleet.spec, operatingSystem: spec.operatingSystem, architecture: spec.architecture } };
  return {
    spec: {
      ...fleet.spec,
      cpuMillicores: spec.cpuMillicores,
      memoryMb: spec.memoryMb,
      diskGb: spec.diskGb,
      capabilities: spec.capabilities,
    },
  };
}

export function useFleetConfiguration({
  organizationId,
  fleet,
  section,
  onSaved,
  refresh,
}: {
  organizationId: string;
  fleet?: SuperplaneOrganizationsFleet;
  section?: FleetSection;
  onSaved: (fleetId: string) => void;
  refresh: () => void;
}) {
  const [draft, setDraft] = useState(() => draftFromFleet(fleet));
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);
  const save = async () => {
    if (saving) return;
    const request = new AbortController();
    abort.current = request;
    setSaving(true);
    setError("");
    const fleetId = fleet?.id ?? draft.fleetId.trim();
    try {
      if (fleet && section) {
        await organizationsUpdateRunnerFleet(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId, fleetId },
            body: sectionPatch(draft, fleet, section),
            signal: request.signal,
          }),
        );
      } else {
        await organizationsCreateRunnerFleet(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId },
            body: { fleetId, enabled: true, runnerVersion: draft.runnerVersion.trim(), spec: fleetSpec(draft) },
            signal: request.signal,
          }),
        );
      }
      if (!request.signal.aborted) {
        refresh();
        onSaved(fleetId);
      }
    } catch (failure) {
      if (request.signal.aborted) return;
      const code = typeof failure === "object" && failure !== null && "code" in failure ? failure.code : undefined;
      setError(
        code === 6
          ? "This fleet ID already exists. Choose a different ID."
          : "Could not save the fleet. Check access and settings, then retry.",
      );
      refresh();
    } finally {
      if (!request.signal.aborted) setSaving(false);
    }
  };
  return { draft, setDraft, error, saving, save };
}
