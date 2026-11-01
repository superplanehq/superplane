import { useEffect, useRef, useState, type FormEvent } from "react";
import {
  organizationsCreateRunnerFleet,
  organizationsUpdateRunnerFleet,
  type SuperplaneOrganizationsFleet,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

function useFleetForm({
  organizationId,
  fleet,
  onSaved,
  refresh,
}: {
  organizationId: string;
  fleet?: SuperplaneOrganizationsFleet;
  onSaved: (fleetId: string) => void;
  refresh: () => void;
}) {
  const [fleetId, setFleetId] = useState(fleet?.id ?? "");
  const [runnerVersion, setRunnerVersion] = useState(fleet?.runnerVersion ?? "");
  const [enabled, setEnabled] = useState(fleet?.enabled ?? true);
  const [spec, setSpec] = useState(fleet?.spec ?? {});
  const [capabilities, setCapabilities] = useState(fleet?.spec?.capabilities?.join(", ") ?? "");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (saving) return;
    const request = new AbortController();
    abort.current = request;
    setSaving(true);
    setError("");
    const body = {
      enabled,
      runnerVersion: runnerVersion.trim(),
      spec: {
        operatingSystem: spec.operatingSystem?.trim(),
        architecture: spec.architecture?.trim(),
        cpuMillicores: spec.cpuMillicores,
        memoryMb: spec.memoryMb,
        diskGb: spec.diskGb,
        capabilities: capabilities
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean),
      },
    };
    try {
      if (fleet) {
        await organizationsUpdateRunnerFleet(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId, fleetId },
            body,
            signal: request.signal,
          }),
        );
      } else {
        await organizationsCreateRunnerFleet(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId },
            body: { ...body, fleetId: fleetId.trim() },
            signal: request.signal,
          }),
        );
      }
      if (!request.signal.aborted) {
        refresh();
        onSaved(fleetId.trim());
      }
    } catch (failure) {
      if (request.signal.aborted) return;
      const code = typeof failure === "object" && failure !== null && "code" in failure ? failure.code : undefined;
      setError(
        code === 6
          ? "This fleet ID already exists. Choose a different ID."
          : "Could not save the fleet. Check access and specification, then retry.",
      );
      refresh();
    } finally {
      if (!request.signal.aborted) setSaving(false);
    }
  };

  return {
    fleetId,
    setFleetId,
    runnerVersion,
    setRunnerVersion,
    enabled,
    setEnabled,
    spec,
    setSpec,
    capabilities,
    setCapabilities,
    error,
    saving,
    save,
  };
}

export function FleetForm(props: {
  organizationId: string;
  fleet?: SuperplaneOrganizationsFleet;
  onSaved: (fleetId: string) => void;
  onCancel: () => void;
  refresh: () => void;
}) {
  const { fleet, onCancel } = props;
  const {
    fleetId,
    setFleetId,
    runnerVersion,
    setRunnerVersion,
    enabled,
    setEnabled,
    spec,
    setSpec,
    capabilities,
    setCapabilities,
    error,
    saving,
    save,
  } = useFleetForm(props);
  return (
    <form onSubmit={save} className="space-y-4 rounded-md border p-4">
      <h2 className="text-lg font-medium">{fleet ? "Edit fleet" : "Create fleet"}</h2>
      <Label htmlFor="fleet-id">Fleet ID</Label>
      <Input
        id="fleet-id"
        required
        value={fleetId}
        disabled={!!fleet || saving}
        onChange={(event) => setFleetId(event.target.value)}
      />
      <Label htmlFor="runner-version">Runner version</Label>
      <Input
        id="runner-version"
        required
        value={runnerVersion}
        onChange={(event) => setRunnerVersion(event.target.value)}
      />
      {(
        [
          ["operatingSystem", "Operating system"],
          ["architecture", "Architecture"],
        ] as const
      ).map(([field, label]) => (
        <div key={field}>
          <Label htmlFor={field}>{label}</Label>
          <Input
            id={field}
            required
            value={spec[field] ?? ""}
            onChange={(event) => setSpec({ ...spec, [field]: event.target.value })}
          />
        </div>
      ))}
      {(
        [
          ["cpuMillicores", "CPU millicores"],
          ["memoryMb", "Memory MB"],
          ["diskGb", "Disk GB"],
        ] as const
      ).map(([field, label]) => (
        <div key={field}>
          <Label htmlFor={field}>{label}</Label>
          <Input
            id={field}
            type="number"
            required
            min={1}
            step={1}
            value={spec[field] ?? ""}
            onChange={(event) => setSpec({ ...spec, [field]: Number(event.target.value) })}
          />
        </div>
      ))}
      <Label htmlFor="capabilities">Capabilities (comma-separated)</Label>
      <Input id="capabilities" value={capabilities} onChange={(event) => setCapabilities(event.target.value)} />
      <div className="flex items-center gap-2">
        <Switch id="fleet-enabled" checked={enabled} onCheckedChange={setEnabled} />
        <Label htmlFor="fleet-enabled">Enabled</Label>
      </div>
      <p className="text-sm text-muted-foreground">
        Disabling stops new work. Active work, inspection, and cleanup remain available.
      </p>
      {error && <p role="alert">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" disabled={saving}>
          {saving ? "Saving…" : "Save fleet"}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
