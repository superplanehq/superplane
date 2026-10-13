import { useState } from "react";

import { Textarea } from "@/components/ui/textarea";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

/** Visual only. The text stays in this field and is not saved. */
export function WorkspaceSetupStepsSection({
  canUpdate,
  commands = "",
}: {
  canUpdate: boolean;
  commands?: string;
}) {
  const [value, setValue] = useState(commands);

  return (
    <FactorySettingsCard
      title={AGENT_RESOURCES_COPY.setupStepsTitle}
      description={AGENT_RESOURCES_COPY.setupStepsHelper}
      data-testid="agent-setup-steps"
    >
      <Textarea
        value={value}
        onChange={(event) => setValue(event.target.value)}
        disabled={!canUpdate}
        rows={6}
        spellCheck={false}
        placeholder={AGENT_RESOURCES_COPY.setupStepsPlaceholder}
        aria-label={AGENT_RESOURCES_COPY.setupStepsTitle}
        className="min-h-28 font-mono text-[13px] leading-5"
        data-testid="agent-setup-steps-commands"
      />
    </FactorySettingsCard>
  );
}
