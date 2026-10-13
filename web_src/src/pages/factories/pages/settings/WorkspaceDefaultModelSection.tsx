import { useState } from "react";

import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

const MODEL_FIELD_ID = "agent-default-model-select";

export type AgentModelChoice = {
  value: string;
  label: string;
};

/** Visual choices only. The selection is not saved. */
export function WorkspaceDefaultModelSection({
  canUpdate,
  models,
}: {
  canUpdate: boolean;
  models: AgentModelChoice[];
}) {
  const [value, setValue] = useState("");

  return (
    <FactorySettingsCard data-testid="agent-default-model">
      <div className="mb-3 flex flex-col gap-1">
        <Label htmlFor={MODEL_FIELD_ID} className="cursor-pointer text-[13px] font-medium tracking-[-0.01em]">
          {AGENT_RESOURCES_COPY.defaultModelTitle}
        </Label>
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.defaultModelHelper}</p>
      </div>
      <Select value={value || undefined} onValueChange={setValue} disabled={!canUpdate}>
        <SelectTrigger id={MODEL_FIELD_ID} className="w-full" data-testid={MODEL_FIELD_ID}>
          <SelectValue placeholder={AGENT_RESOURCES_COPY.defaultModelPlaceholder} />
        </SelectTrigger>
        <SelectContent>
          {models.map((model) => (
            <SelectItem key={model.value} value={model.value}>
              {model.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FactorySettingsCard>
  );
}
