import { Bot, ChevronDown } from "lucide-react";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import { modelNameWithThinking, supportedThinkingLevel, thinkingOptions } from "@/lib/thinkingLevel";

import { DRAFT_START_MODEL_AUTO, draftStartThinkingLevels } from "./draftStartModel";

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

/**
 * Picks the runner model and thinking for Start. `icon` is the chevron fused to Start,
 * `labeled` the capsule segment, `ghost` the quiet control on the refine
 * strip settings row. Closed labeled and ghost triggers show the model name
 * and the thinking word when one is chosen. Thinking lists the levels of the
 * selected model and is hidden for Auto or a model without levels.
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
  const runnerModels = models.data?.models;
  const selectedName =
    model === DRAFT_START_MODEL_AUTO ? "Auto" : (runnerModels ?? []).find((item) => item.id === model)?.name || model;
  const levels = draftStartThinkingLevels(runnerModels, model);
  const thinking = thinkingOptions(levels, { withDefault: false });
  const selectedThinking = supportedThinkingLevel(thinkingLevel, levels);
  const closedLabel = modelNameWithThinking(selectedName, selectedThinking);
  const catalogLoaded = runnerModels !== undefined;

  // A refreshed catalog can drop the chosen level. Clear it so Start never sends a hidden choice.
  useEffect(() => {
    if (catalogLoaded && thinkingLevel !== selectedThinking) {
      onChange({ model, thinkingLevel: selectedThinking });
    }
  }, [catalogLoaded, model, thinkingLevel, selectedThinking, onChange]);
  const showName = appearance !== "icon";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size={showName ? "sm" : "icon-xs"}
          variant={TRIGGER_VARIANT[appearance]}
          aria-label={`Model: ${closedLabel}`}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className={TRIGGER_CLASS[appearance]}
        >
          {appearance === "ghost" ? <Bot className="size-4" aria-hidden /> : null}
          {showName ? closedLabel : null}
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
            ...(runnerModels ?? [])
              .filter((item) => (item.id ?? "") !== "")
              .map((item) => ({ value: item.id ?? "", label: item.name || item.id || "" })),
          ]}
          onValueChange={(nextModel) =>
            onChange({
              model: nextModel,
              thinkingLevel: supportedThinkingLevel(thinkingLevel, draftStartThinkingLevels(runnerModels, nextModel)),
            })
          }
        />
        {thinking.length > 0 ? (
          <DropdownMenuValueSub
            label="Thinking"
            testId="split-run-draft-thinking"
            value={selectedThinking}
            options={thinking}
            onValueChange={(nextThinking) => onChange({ model, thinkingLevel: nextThinking })}
          />
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
