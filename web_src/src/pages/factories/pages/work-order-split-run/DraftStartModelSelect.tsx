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
  appearance = "icon",
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  value: string;
  onChange: (next: string) => void;
  disabled?: boolean;
  appearance?: "icon" | "labeled";
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName);
  const selectedName =
    value === DRAFT_START_MODEL_AUTO ? "Auto" : (models.data ?? []).find((model) => model.id === value)?.name || value;
  const labeled = appearance === "labeled";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size={labeled ? "sm" : "icon-xs"}
          variant={labeled ? "outline" : "default"}
          aria-label={`Model: ${selectedName}`}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className={
            labeled
              ? "!rounded-none h-7 gap-1.5 border-0 bg-background text-xs shadow-none"
              : "rounded-md rounded-l-none"
          }
        >
          {labeled ? selectedName : null}
          <ChevronDown className={labeled ? "size-3 opacity-60" : "size-3.5"} aria-hidden />
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
