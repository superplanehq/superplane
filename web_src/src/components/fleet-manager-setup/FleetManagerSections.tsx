import { useId } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";

import { AwsProviderLogo, AzureProviderLogo, GcpProviderLogo } from "./providerLogos";

export type FleetManagerSetupState = ReturnType<typeof useFleetManagerSetup>;

const LOADED_FROM_TERRAFORM = "Loaded from Terraform";

const Field = ({
  id,
  label,
  imported = false,
  ...props
}: { id: string; label: string; imported?: boolean } & React.ComponentProps<typeof Input>) => (
  <div className="space-y-1.5 text-sm">
    <div className="flex items-center justify-between gap-3">
      <Label htmlFor={id}>{label}</Label>
      {imported && <span className="text-[11px] text-emerald-400">{LOADED_FROM_TERRAFORM}</span>}
    </div>
    <Input id={id} {...props} />
  </div>
);

const PROVIDERS = [
  { id: "aws", name: "AWS", Logo: AwsProviderLogo },
  { id: "gcp", name: "GCP", Logo: GcpProviderLogo },
  { id: "azure", name: "Azure", Logo: AzureProviderLogo },
] as const;

function ProviderCard({
  id,
  name,
  Logo,
  selected,
  loadedFromTerraform,
  noteId,
  onSelect,
}: {
  id: string;
  name: string;
  Logo: typeof AwsProviderLogo;
  selected: boolean;
  loadedFromTerraform: boolean;
  noteId: string;
  onSelect: () => void;
}) {
  const card = (
    <Button
      type="button"
      variant={selected ? "default" : "outline"}
      className="aspect-square h-auto w-full rounded-xl p-0"
      aria-label={name}
      aria-describedby={loadedFromTerraform ? noteId : undefined}
      aria-pressed={selected}
      onClick={onSelect}
    >
      <Logo className="size-[calc(100%-1rem)]" testId={`provider-logo-${id}`} />
      {loadedFromTerraform && (
        <span id={noteId} className="sr-only">
          {LOADED_FROM_TERRAFORM}
        </span>
      )}
    </Button>
  );

  if (!loadedFromTerraform) return card;

  return (
    <Tooltip>
      <TooltipTrigger asChild>{card}</TooltipTrigger>
      <TooltipContent side="bottom">{LOADED_FROM_TERRAFORM}</TooltipContent>
    </Tooltip>
  );
}

export function FleetManagerProviderSection({ setup }: { setup: FleetManagerSetupState }) {
  const terraformNoteIdPrefix = useId();
  if (!setup.metadata || !setup.draft) return null;
  const terraformProvider = setup.metadata.infrastructureDefaults.provider;

  return (
    <div className="grid grid-cols-3 gap-3" role="radiogroup" aria-label="Cloud provider">
      {PROVIDERS.map(({ id, name, Logo }) => (
        <ProviderCard
          key={id}
          id={id}
          name={name}
          Logo={Logo}
          selected={setup.draft?.provider === id}
          loadedFromTerraform={terraformProvider === id}
          noteId={`${terraformNoteIdPrefix}-${id}`}
          onSelect={() => setup.setProvider(id)}
        />
      ))}
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
  const metadata = setup.metadata;
  const draft = setup.draft;

  return (
    <div>
      <div className={contained ? "max-h-52 space-y-2 overflow-y-auto pr-1" : "space-y-2"}>
        {metadata.fleetPresets.map((preset) => {
          const id = `fleet-${preset.id}`;
          const selected = draft.selectedFleetIds.includes(preset.id);
          const blocked =
            metadata.infrastructureDefaults.source === "terraform" && !setup.hasFleetImage(preset.spec.architecture);
          return (
            <div key={preset.id} className="flex items-start gap-3 rounded-md border border-border p-3">
              <Checkbox
                id={id}
                checked={selected}
                disabled={blocked && !selected}
                onChange={(event) => setup.toggleFleet(preset.id, event.target.checked)}
              />
              <div className="min-w-0 flex-1 space-y-3">
                <Label htmlFor={id} className="text-sm">
                  <span className="block font-medium">{preset.id}</span>
                  <span className="block text-xs text-muted-foreground">
                    {preset.spec.architecture} · {preset.spec.cpuMillicores}m CPU · {preset.spec.memoryMb} MB memory
                  </span>
                  {blocked && (
                    <span className="block text-xs text-amber-600 dark:text-amber-300">
                      Add the {preset.spec.architecture} runner image through Terraform to use this fleet.
                    </span>
                  )}
                </Label>
                {selected && (
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Field
                      id={`machine-type-${preset.id}`}
                      label={`Machine type for ${preset.id}`}
                      value={draft.fields[`machineType.${preset.id}`] || ""}
                      onChange={(event) => setup.setField(`machineType.${preset.id}`, event.target.value)}
                    />
                    <Field
                      id={`warm-capacity-${preset.id}`}
                      label={`Warm capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={draft.fleetCapacities[preset.id]?.warmCapacity ?? 0}
                      onChange={(event) =>
                        setup.setFleetCapacity(preset.id, { warmCapacity: Number(event.target.value) })
                      }
                    />
                    <Field
                      id={`max-capacity-${preset.id}`}
                      label={`Maximum capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={draft.fleetCapacities[preset.id]?.maxCapacity ?? 1}
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
        <Select value={draft.runnerVersion} onValueChange={setup.setRunnerVersion}>
          <SelectTrigger id="runner-version" className="mt-1.5 w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {metadata.runnerVersions.map((version) => (
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
