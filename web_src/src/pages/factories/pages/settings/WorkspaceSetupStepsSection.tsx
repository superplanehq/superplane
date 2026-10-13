import { useState } from "react";

import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

const COMMANDS_FIELD_ID = "agent-setup-steps-commands";

/** Visual only. The text stays in this field and is not saved. */
export function WorkspaceSetupStepsSection({ canUpdate, commands = "" }: { canUpdate: boolean; commands?: string }) {
  const [value, setValue] = useState(commands);

  return (
    <FactorySettingsCard data-testid="agent-setup-steps">
      <div className="mb-3 flex flex-col gap-1">
        <Label htmlFor={COMMANDS_FIELD_ID} className="cursor-pointer text-[13px] font-medium tracking-[-0.01em]">
          {AGENT_RESOURCES_COPY.setupStepsTitle}
        </Label>
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.setupStepsHelper}</p>
      </div>
      <Textarea
        id={COMMANDS_FIELD_ID}
        value={value}
        onChange={(event) => setValue(event.target.value)}
        disabled={!canUpdate}
        rows={6}
        spellCheck={false}
        placeholder={AGENT_RESOURCES_COPY.setupStepsPlaceholder}
        className="min-h-28 font-mono text-[13px] leading-5"
        data-testid={COMMANDS_FIELD_ID}
      />
    </FactorySettingsCard>
  );
}
