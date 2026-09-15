import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";

import { DRAFT_START_MODEL_AUTO, DRAFT_START_MODEL_HELP } from "./draftStartModel";

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
  const [pickerOpen, setPickerOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);

  return (
    <DropdownMenu
      open={pickerOpen}
      onOpenChange={(open) => {
        setPickerOpen(open);
        if (open) {
          setHelpOpen(false);
        }
      }}
    >
      <HoverCard
        open={helpOpen && !pickerOpen}
        onOpenChange={(open) => {
          if (!pickerOpen) {
            setHelpOpen(open);
          }
        }}
        openDelay={150}
        closeDelay={100}
      >
        <HoverCardTrigger asChild>
          <div className="inline-flex h-full" data-testid="split-run-draft-model-wrap">
            <DropdownMenuTrigger asChild disabled={disabled}>
              <Button
                type="button"
                size="icon-xs"
                variant="default"
                aria-label="Model"
                data-testid="split-run-draft-model"
                disabled={disabled}
                className="rounded-l-none border-l border-primary-foreground/25"
              >
                <ChevronDown className="size-3.5" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
          </div>
        </HoverCardTrigger>
        <HoverCardContent side="top" align="end" className="pointer-events-none w-64 space-y-1 p-3 text-sm">
          <p data-testid="split-run-draft-model-help">{DRAFT_START_MODEL_HELP[0]}</p>
          <p>{DRAFT_START_MODEL_HELP[1]}</p>
        </HoverCardContent>
      </HoverCard>
      <DropdownMenuContent align="end" className="max-h-60">
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
