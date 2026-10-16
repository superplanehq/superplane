import { useId } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { DEFAULT_SETUP_FLEET_ID, formatMachineTypeOption, machineTypeChoices } from "@/lib/fleetManagerSetup";

import { AwsProviderLogo, AzureProviderLogo, GcpProviderLogo } from "./providerLogos";

export type FleetManagerSetupState = ReturnType<typeof useFleetManagerSetup>;

const LOADED_FROM_TERRAFORM = "Loaded from Terraform";

const FLEET_HELPER = "Default automations use this fleet. You can add more fleets later.";
const WARM_CAPACITY_HELP =
  "Idle machines kept ready before a task starts. Use 0 to start a machine only when work arrives.";
const MAX_CAPACITY_HELP = "The most machines this fleet can run at one time, including busy machines.";

const CapacityField = ({
  id,
  label,
  description,
  ...props
}: { id: string; label: string; description: string } & React.ComponentProps<typeof Input>) => (
  <div className="flex h-full flex-col gap-1.5 text-sm">
    <Label htmlFor={id}>{label}</Label>
    <p id={`${id}-help`} className="text-xs leading-5 text-muted-foreground">
      {description}
    </p>
    <Input id={id} aria-describedby={`${id}-help`} className="mt-auto" {...props} />
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
      className="relative aspect-[5/3]! h-auto! w-full overflow-hidden rounded-xl! p-0! shadow-none!"
      aria-label={name}
      aria-describedby={loadedFromTerraform ? noteId : undefined}
      aria-pressed={selected}
      onClick={onSelect}
    >
      <span className="absolute inset-0 flex items-center justify-center p-5">
        <Logo className="size-auto h-auto max-h-full w-auto max-w-full" testId={`provider-logo-${id}`} />
      </span>
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

export function FleetManagerFleetSection({ setup }: { setup: FleetManagerSetupState }) {
  const baseId = useId();
  if (!setup.metadata || !setup.draft) return null;
  const metadata = setup.metadata;
  const draft = setup.draft;
  const preset = metadata.fleetPresets.find((item) => item.id === DEFAULT_SETUP_FLEET_ID);
  if (!preset) {
    return (
      <p role="alert" className="text-sm text-destructive">
        SuperPlane could not find the default fleet.
      </p>
    );
  }

  const fleetFieldId = `${baseId}-fleet`;
  const machineTypeFieldId = `${baseId}-machine-type`;
  const warmFieldId = `${baseId}-warm`;
  const maxFieldId = `${baseId}-max`;
  const machineTypeKey = `machineType.${DEFAULT_SETUP_FLEET_ID}`;
  const machineType = draft.fields[machineTypeKey] || "";
  const blocked =
    metadata.infrastructureDefaults.source === "terraform" && !setup.hasFleetImage(preset.spec.architecture);
  const capacity = draft.fleetCapacities[DEFAULT_SETUP_FLEET_ID];

  return (
    <div className="space-y-4">
      <div className="space-y-1.5 text-sm">
        <Label htmlFor={fleetFieldId}>Fleet</Label>
        <p id={`${fleetFieldId}-help`} className="text-xs leading-5 text-muted-foreground">
          {FLEET_HELPER}
        </p>
        <Input id={fleetFieldId} value={DEFAULT_SETUP_FLEET_ID} readOnly aria-describedby={`${fleetFieldId}-help`} />
      </div>
      {blocked && (
        <p className="text-xs text-amber-600 dark:text-amber-300">
          Add the {preset.spec.architecture} runner image through Terraform to use this fleet.
        </p>
      )}
      <div className="space-y-1.5 text-sm">
        <Label htmlFor={machineTypeFieldId}>Machine type</Label>
        <Select value={machineType || undefined} onValueChange={(value) => setup.setField(machineTypeKey, value)}>
          <SelectTrigger id={machineTypeFieldId} className="w-full">
            <SelectValue placeholder="Select a machine type" />
          </SelectTrigger>
          <SelectContent>
            {machineTypeChoices(draft.provider, machineType).map((option) => (
              <SelectItem key={option.id} value={option.id}>
                {formatMachineTypeOption(option)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {machineType && (
        <div className="grid items-stretch gap-3 sm:grid-cols-2">
          <CapacityField
            id={warmFieldId}
            label="Warm capacity"
            description={WARM_CAPACITY_HELP}
            type="number"
            min={0}
            value={capacity?.warmCapacity ?? 0}
            onChange={(event) =>
              setup.setFleetCapacity(DEFAULT_SETUP_FLEET_ID, { warmCapacity: Number(event.target.value) })
            }
          />
          <CapacityField
            id={maxFieldId}
            label="Maximum capacity"
            description={MAX_CAPACITY_HELP}
            type="number"
            min={0}
            value={capacity?.maxCapacity ?? 1}
            onChange={(event) =>
              setup.setFleetCapacity(DEFAULT_SETUP_FLEET_ID, { maxCapacity: Number(event.target.value) })
            }
          />
        </div>
      )}
      {setup.fleetErrors.length > 0 && (
        <p role="alert" className="text-sm text-destructive">
          {setup.fleetErrors[0]}
        </p>
      )}
    </div>
  );
}
