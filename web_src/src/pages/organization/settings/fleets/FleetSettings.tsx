import { useEffect, useState, type FormEvent } from "react";
import type { SuperplaneOrganizationsFleet } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useFleetConfiguration } from "@/hooks/useFleetConfiguration";
import type { FleetSection } from "@/lib/organizationFleetConfiguration";
import { FleetConfigurationFields } from "./FleetConfigurationFields";

const sections = [
  { id: "version", title: "Runner version", description: "Runner release used by this fleet." },
  { id: "platform", title: "Platform", description: "Operating system and architecture required for runners." },
  { id: "resources", title: "Runner resources", description: "Resources and capabilities that runners must provide." },
] as const;

export function FleetSettings({
  organizationId,
  fleet,
  canConfigure,
  refresh,
}: {
  organizationId: string;
  fleet: SuperplaneOrganizationsFleet;
  canConfigure: boolean;
  refresh: () => void;
}) {
  const [editing, setEditing] = useState<FleetSection | null>(null);
  useEffect(() => {
    if (!canConfigure) setEditing(null);
  }, [canConfigure]);
  const spec = fleet.spec ?? {};
  const values = {
    version: { "Runner version": fleet.runnerVersion },
    platform: { "Operating system": spec.operatingSystem, Architecture: spec.architecture },
    resources: {
      "CPU millicores": spec.cpuMillicores,
      "Memory MB": spec.memoryMb,
      "Disk GB": spec.diskGb,
      Capabilities: spec.capabilities?.join(", ") || "None",
    },
  };
  return (
    <div className="space-y-4">
      {sections.map(({ id, title, description }) => (
        <section key={id} className="rounded-lg border border-border bg-card p-4">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div>
              <h3 className="font-medium">{title}</h3>
              <p className="mt-1 text-[12px] text-muted-foreground">{description}</p>
            </div>
            {canConfigure && editing !== id && (
              <Button
                size="xs"
                variant="outline"
                disabled={editing !== null}
                aria-label={`Change ${title.toLowerCase()}`}
                onClick={() => setEditing(id)}
              >
                Change
              </Button>
            )}
          </div>
          {editing === id && canConfigure ? (
            <FleetSectionForm
              organizationId={organizationId}
              fleet={fleet}
              section={id}
              refresh={refresh}
              onClose={() => setEditing(null)}
            />
          ) : (
            <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3">
              {Object.entries(values[id]).map(([label, value]) => (
                <div key={label}>
                  <dt className="text-[12px] text-muted-foreground">{label}</dt>
                  <dd className="mt-1">{value ?? "Unavailable"}</dd>
                </div>
              ))}
            </dl>
          )}
        </section>
      ))}
    </div>
  );
}

function FleetSectionForm({
  organizationId,
  fleet,
  section,
  refresh,
  onClose,
}: {
  organizationId: string;
  fleet: SuperplaneOrganizationsFleet;
  section: FleetSection;
  refresh: () => void;
  onClose: () => void;
}) {
  const { draft, setDraft, error, saving, save } = useFleetConfiguration({
    organizationId,
    fleet,
    section,
    onSaved: onClose,
    refresh,
  });
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void save();
  };
  return (
    <form onSubmit={submit} className="space-y-4">
      <FleetConfigurationFields section={section} draft={draft} onChange={setDraft} disabled={saving} />
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <div className="flex justify-end gap-2 border-t border-border pt-3">
        <Button type="button" size="sm" variant="outline" onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button type="submit" size="sm" disabled={saving}>
          {saving ? "Saving…" : "Save changes"}
        </Button>
      </div>
    </form>
  );
}
