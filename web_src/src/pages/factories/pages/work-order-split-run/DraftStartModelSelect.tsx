import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
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
  const selectedName =
    value === DRAFT_START_MODEL_AUTO ? "Auto" : (models.data ?? []).find((model) => model.id === value)?.name || value;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size="icon-xs"
          variant="default"
          aria-label={`Model: ${selectedName}`}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className="rounded-md rounded-l-none"
        >
          <ChevronDown className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="min-w-44">
        <DropdownMenuRadioGroup value={value} onValueChange={onChange}>
          <DropdownMenuRadioItem value={DRAFT_START_MODEL_AUTO}>Auto</DropdownMenuRadioItem>
          {(models.data ?? []).map((model) => {
            const id = model.id ?? "";
            if (id === "") {
              return null;
            }
            return (
              <DropdownMenuRadioItem key={id} value={id}>
                {model.name || id}
              </DropdownMenuRadioItem>
            );
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
