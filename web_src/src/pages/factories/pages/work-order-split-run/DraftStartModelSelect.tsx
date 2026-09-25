import { Bot, ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import { DRAFT_START_THINKING_AUTO, DRAFT_START_THINKING_DEFAULT, THINKING_LEVELS } from "@/lib/thinkingLevel";

import { DRAFT_START_MODEL_AUTO } from "./draftStartModel";

type Appearance = "icon" | "labeled" | "ghost";

const TRIGGER_VARIANT: Record<Appearance, "default" | "outline" | "ghost"> = {
  icon: "default",
  labeled: "outline",
  ghost: "ghost",
};

const TRIGGER_CLASS: Record<Appearance, string> = {
  icon: "rounded-md rounded-l-none",
  labeled: "!rounded-none h-7 gap-1.5 border-0 bg-background text-xs shadow-none",
  ghost: "gap-1.5 text-muted-foreground hover:text-foreground",
};

const START_THINKING_LEVELS = [
  { value: DRAFT_START_THINKING_AUTO, label: "Auto" },
  ...THINKING_LEVELS.map((level) => ({
    value: level.value === "" ? DRAFT_START_THINKING_DEFAULT : level.value,
    label: level.label,
  })),
];

/**
 * Picks the runner model and thinking for Start. `icon` is the chevron fused to Start,
 * `labeled` the capsule segment, `ghost` the quiet control on the refine
 * strip settings row. Closed labeled and ghost triggers show the model name.
 */
export function DraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  model,
  thinkingLevel,
  onChange,
  disabled = false,
  appearance = "icon",
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  model: string;
  thinkingLevel: string;
  onChange: (next: { model: string; thinkingLevel: string }) => void;
  disabled?: boolean;
  appearance?: Appearance;
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName);
  const selectedName =
    model === DRAFT_START_MODEL_AUTO ? "Auto" : (models.data ?? []).find((item) => item.id === model)?.name || model;
  const showName = appearance !== "icon";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size={showName ? "sm" : "icon-xs"}
          variant={TRIGGER_VARIANT[appearance]}
          aria-label={`Model: ${selectedName}`}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className={TRIGGER_CLASS[appearance]}
        >
          {appearance === "ghost" ? <Bot className="size-4" aria-hidden /> : null}
          {showName ? selectedName : null}
          <ChevronDown className={showName ? "size-3 opacity-60" : "size-3.5"} aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="min-w-44">
        <DropdownMenuValueSub
          label="Model"
          testId="split-run-draft-model-list"
          value={model}
          options={[
            { value: DRAFT_START_MODEL_AUTO, label: "Auto" },
            ...(models.data ?? [])
              .filter((item) => (item.id ?? "") !== "")
              .map((item) => ({ value: item.id ?? "", label: item.name || item.id || "" })),
          ]}
          onValueChange={(nextModel) => onChange({ model: nextModel, thinkingLevel })}
        />
        <DropdownMenuValueSub
          label="Thinking"
          testId="split-run-draft-thinking"
          value={thinkingLevel}
          options={START_THINKING_LEVELS}
          onValueChange={(nextThinking) => onChange({ model, thinkingLevel: nextThinking })}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
