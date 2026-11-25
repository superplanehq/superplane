import { Bot, ChevronDown } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/ui/dropdownMenu";
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";

import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import { THINKING_LEVEL_HIGH, THINKING_LEVEL_LOW, THINKING_LEVEL_MEDIUM } from "@/lib/thinkingLevel";

import {
  draftStartDefaultLabels,
  draftStartModelOptions,
  resolveDraftStartSelection,
  type DraftStartSelection,
} from "./draftStartModel";
import { draftStartDefaultsStatus, type DraftStartDefaults } from "@/hooks/useDraftStartDefaults";
import { DraftStartSelectionLabel } from "./DraftStartSelectionLabel";

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
  { value: THINKING_LEVEL_LOW, label: "Low" },
  { value: THINKING_LEVEL_MEDIUM, label: "Medium" },
  { value: THINKING_LEVEL_HIGH, label: "High" },
];

/**
 * Picks the runner model and thinking for Start. Use the saved owner's
 * catalog, or the caller's catalog when the task has no owner.
 * `icon` is the chevron fused to Start,
 * `labeled` the capsule segment, `ghost` the quiet control on the refine
 * strip settings row. Closed labeled and ghost triggers show the model name
 * and the thinking word when one is chosen.
 */
export function DraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  orderId,
  model,
  thinkingLevel,
  defaults,
  onChange,
  disabled = false,
  appearance = "icon",
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  orderId?: string;
  model?: string;
  thinkingLevel?: string;
  defaults: DraftStartDefaults;
  onChange: (next: DraftStartSelection) => void;
  disabled?: boolean;
  appearance?: Appearance;
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName, true, orderId);
  const catalog = models.data?.models;
  const modelOptions = useMemo(() => draftStartModelOptions(catalog), [catalog]);
  const resolved = resolveDraftStartSelection(
    { model, thinkingLevel },
    defaults.values ?? {},
    modelOptions,
    defaults.values ? undefined : draftStartDefaultsStatus(defaults),
  );
  const showName = appearance !== "icon";
  const defaultLabels = draftStartDefaultLabels(defaults.values, modelOptions);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size={showName ? "sm" : "icon-xs"}
          variant={TRIGGER_VARIANT[appearance]}
          aria-label={resolved.accessibleLabel}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className={`${TRIGGER_CLASS[appearance]} min-w-0 max-w-full`}
        >
          {appearance === "ghost" ? <Bot className="size-4" aria-hidden /> : null}
          {showName ? <DraftStartSelectionLabel resolved={resolved} thinkingLevel={thinkingLevel} /> : null}
          <ChevronDown className={showName ? "size-3 opacity-60" : "size-3.5"} aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="min-w-44 max-w-[calc(100vw-1rem)]">
        {!defaults.values ? (
          <DropdownMenuItem onSelect={defaults.retry}>Retry Implementation defaults</DropdownMenuItem>
        ) : null}
        <DropdownMenuValueSub
          label="Model"
          testId="split-run-draft-model-list"
          value={model ?? ""}
          displayedValue={resolved.modelLabel}
          options={[{ value: "", label: defaultLabels.model }, ...modelOptions]}
          onValueChange={(nextModel) => onChange({ model: nextModel || undefined, thinkingLevel })}
        />
        <DropdownMenuValueSub
          label="Thinking"
          testId="split-run-draft-thinking"
          value={thinkingLevel ?? ""}
          displayedValue={resolved.thinkingLabel}
          options={[{ value: "", label: defaultLabels.thinking }, ...START_THINKING_LEVELS]}
          onValueChange={(nextThinking) => onChange({ model, thinkingLevel: nextThinking || undefined })}
        />
        {resolved.hasOverrides ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onChange({})}>Use defaults</DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
