import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { Cloud } from "lucide-react";

export type FleetManagerSetupState = ReturnType<typeof useFleetManagerSetup>;

const Field = ({
  id,
  label,
  imported = false,
  ...props
}: { id: string; label: string; imported?: boolean } & React.ComponentProps<typeof Input>) => (
  <div className="space-y-1.5 text-sm">
    <div className="flex items-center justify-between gap-3">
      <Label htmlFor={id}>{label}</Label>
      {imported && <span className="text-[11px] text-emerald-400">Loaded from Terraform</span>}
    </div>
    <Input id={id} {...props} />
  </div>
);

export function FleetManagerProviderSection({ setup }: { setup: FleetManagerSetupState }) {
  if (!setup.metadata || !setup.draft) return null;
  const terraformProvider = setup.metadata.infrastructureDefaults.provider;

  return (
    <div className="grid gap-3 sm:grid-cols-3" role="radiogroup" aria-label="Cloud provider">
      {(["aws", "gcp", "azure"] as const).map((provider) => {
        const selected = setup.draft?.provider === provider;
        return (
          <Button
            key={provider}
            type="button"
            variant={selected ? "default" : "outline"}
            className="h-auto min-h-20 flex-col gap-1"
            aria-pressed={selected}
            onClick={() => setup.setProvider(provider)}
          >
            <Cloud className="size-4" aria-hidden />
            <span>{provider.toUpperCase()}</span>
            {terraformProvider === provider && <span className="text-[10px] normal-case">Loaded from Terraform</span>}
          </Button>
        );
      })}
    </div>
  );
}

export function FleetManagerFleetSection({
  setup,
  contained = false,
}: {
  setup: FleetManagerSetupState;
  contained?: boolean;
}) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div>
      <div className={contained ? "max-h-52 space-y-2 overflow-y-auto pr-1" : "space-y-2"}>
        {setup.metadata.fleetPresets.map((preset) => {
          const id = `fleet-${preset.id}`;
          const hasImage = setup.hasFleetImage(preset.spec.architecture);
          return (
            <div key={preset.id} className="flex items-start gap-3 rounded-md border border-border p-3">
              <Checkbox
                id={id}
                checked={setup.draft?.selectedFleetIds.includes(preset.id)}
                disabled={!hasImage}
                onChange={(event) => setup.toggleFleet(preset.id, event.target.checked)}
              />
              <div className="min-w-0 flex-1 space-y-3">
                <Label htmlFor={id} className="text-sm">
                  <span className="block font-medium">{preset.id}</span>
                  <span className="block text-xs text-muted-foreground">
                    {preset.spec.architecture} · {preset.spec.cpuMillicores}m CPU · {preset.spec.memoryMb} MB memory
                  </span>
                  {!hasImage && (
                    <span className="block text-xs text-amber-600 dark:text-amber-300">
                      Add the {preset.spec.architecture} runner image through Terraform to use this fleet.
                    </span>
                  )}
                </Label>
                {setup.draft?.selectedFleetIds.includes(preset.id) && (
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Field
                      id={`machine-type-${preset.id}`}
                      label={`Machine type for ${preset.id}`}
                      value={setup.draft.fields[`machineType.${preset.id}`] || ""}
                      onChange={(event) => setup.setField(`machineType.${preset.id}`, event.target.value)}
                    />
                    <Field
                      id={`warm-capacity-${preset.id}`}
                      label={`Warm capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={setup.draft.fleetCapacities[preset.id]?.warmCapacity ?? 0}
                      onChange={(event) =>
                        setup.setFleetCapacity(preset.id, { warmCapacity: Number(event.target.value) })
                      }
                    />
                    <Field
                      id={`max-capacity-${preset.id}`}
                      label={`Maximum capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={setup.draft.fleetCapacities[preset.id]?.maxCapacity ?? 1}
                      onChange={(event) =>
                        setup.setFleetCapacity(preset.id, { maxCapacity: Number(event.target.value) })
                      }
                    />
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
      <div className="mt-5 max-w-xs">
        <Label htmlFor="runner-version">Runner version</Label>
        <Select value={setup.draft.runnerVersion} onValueChange={setup.setRunnerVersion}>
          <SelectTrigger id="runner-version" className="mt-1.5 w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {setup.metadata.runnerVersions.map((version) => (
              <SelectItem key={version} value={version}>
                {version}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {setup.fleetErrors.length > 0 && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {setup.fleetErrors[0]}
        </p>
      )}
    </div>
  );
}
