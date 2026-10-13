import { useState } from "react";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

/** Visual choices only. The selection is not saved. */
const PREVIEW_MODELS = [
  { value: "claude-sonnet", label: "Claude Sonnet" },
  { value: "claude-opus", label: "Claude Opus" },
  { value: "gpt-5", label: "GPT-5" },
];

export function WorkspaceDefaultModelSection({ canUpdate }: { canUpdate: boolean }) {
  const [value, setValue] = useState("");

  return (
    <FactorySettingsCard
      title={AGENT_RESOURCES_COPY.defaultModelTitle}
      description={AGENT_RESOURCES_COPY.defaultModelHelper}
      data-testid="agent-default-model"
    >
      <Select value={value || undefined} onValueChange={setValue} disabled={!canUpdate}>
        <SelectTrigger
          className="w-full"
          aria-label={AGENT_RESOURCES_COPY.defaultModelTitle}
          data-testid="agent-default-model-select"
        >
          <SelectValue placeholder={AGENT_RESOURCES_COPY.defaultModelPlaceholder} />
        </SelectTrigger>
        <SelectContent>
          {PREVIEW_MODELS.map((model) => (
            <SelectItem key={model.value} value={model.value}>
              {model.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FactorySettingsCard>
  );
}
