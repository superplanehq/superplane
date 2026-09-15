import type { ReactNode } from "react";
import { Check, ChevronDown, Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";

import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";

export function DraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  value,
  onChange,
  disabled = false,
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  value: string;
  onChange: (next: string) => void;
  disabled?: boolean;
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size="icon-xs"
          variant="default"
          aria-label="Model"
          data-testid="split-run-draft-model"
          disabled={disabled}
        >
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="min-w-44 [--radius:1rem]">
        <DropdownMenuGroup>
          <ModelMenuItem
            label="Auto"
            selected={value === DRAFT_START_MODEL_AUTO}
            icon={<Sparkles />}
            onSelect={() => onChange(DRAFT_START_MODEL_AUTO)}
          />
          {(models.data ?? []).map((model) => {
            const id = model.id ?? "";
            if (id === "") {
              return null;
            }
            return (
              <ModelMenuItem key={id} label={model.name || id} selected={value === id} onSelect={() => onChange(id)} />
            );
          })}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ModelMenuItem({
  label,
  selected,
  icon,
  onSelect,
}: {
  label: string;
  selected: boolean;
  icon?: ReactNode;
  onSelect: () => void;
}) {
  return (
    <DropdownMenuItem onSelect={onSelect}>
      {icon}
      {label}
      {selected ? <Check className="ml-auto" aria-hidden /> : null}
    </DropdownMenuItem>
  );
}
