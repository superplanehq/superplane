import { useState } from "react";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

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
    <Select
      value={value}
      onValueChange={onChange}
      disabled={disabled}
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
          <div className="inline-flex" data-testid="split-run-draft-model-wrap">
            <SelectTrigger size="sm" className="w-[11.5rem]" aria-label="Model" data-testid="split-run-draft-model">
              <SelectValue placeholder="Auto" />
            </SelectTrigger>
          </div>
        </HoverCardTrigger>
        <HoverCardContent side="top" align="end" className="pointer-events-none w-64 space-y-1 p-3 text-sm">
          <p data-testid="split-run-draft-model-help">{DRAFT_START_MODEL_HELP[0]}</p>
          <p>{DRAFT_START_MODEL_HELP[1]}</p>
        </HoverCardContent>
      </HoverCard>
      <SelectContent position="popper" className="max-h-60">
        <SelectItem value={DRAFT_START_MODEL_AUTO}>Auto</SelectItem>
        {(models.data ?? []).map((model) => {
          const id = model.id ?? "";
          if (id === "") {
            return null;
          }
          return (
            <SelectItem key={id} value={id}>
              {model.name || id}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}
