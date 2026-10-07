import { Check, ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import {
  THINKING_LEVEL_HIGH,
  THINKING_LEVEL_LOW,
  THINKING_LEVEL_MEDIUM,
  modelNameWithThinking,
} from "@/lib/thinkingLevel";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { DRAFT_START_MODEL_AUTO } from "../pages/work-order-split-run/draftStartModel";

const THINKING_OPTIONS = [
  { value: THINKING_LEVEL_LOW, label: "Low" },
  { value: THINKING_LEVEL_MEDIUM, label: "Medium" },
  { value: THINKING_LEVEL_HIGH, label: "High" },
];

const MENU_ITEM_CLASS = "cursor-pointer text-[13px]";

type StartChoice = {
  model: string;
  thinkingLevel: string;
};

export function MobileDraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  model,
  thinkingLevel,
  onChange,
  disabled = false,
}: {
  organizationId?: string;
  factoryId?: string;
  lineName?: string;
  model: string;
  thinkingLevel: string;
  onChange: (next: StartChoice) => void;
  disabled?: boolean;
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName, Boolean(lineName));
  const modelOptions = lineName ? runnerModelOptions(models.data) : [{ value: DRAFT_START_MODEL_AUTO, label: "Auto" }];
  const selectedName = modelOptions.find((option) => option.value === model)?.label || model;
  const closedLabel = modelNameWithThinking(selectedName, thinkingLevel);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="outline"
          aria-label={`Model: ${closedLabel}`}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className="h-7 max-w-36 min-w-0 shrink gap-1.5 overflow-hidden !rounded-none border-0 bg-background px-2 text-xs shadow-none"
        >
          <span className="min-w-0 truncate">{closedLabel}</span>
          <ChevronDown className="size-3 shrink-0 opacity-60" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        side="bottom"
        collisionPadding={8}
        className="w-[min(18rem,var(--radix-dropdown-menu-content-available-width))] max-w-[calc(100vw-1rem)]"
      >
        <DropdownMenuLabel className="text-xs text-muted-foreground">Model</DropdownMenuLabel>
        <div className="max-h-52 overflow-y-auto" data-testid="mobile-task-model-list">
          {modelOptions.map((option) => (
            <ChoiceItem
              key={option.value}
              label={option.label}
              selected={option.value === model}
              onSelect={() => onChange({ model: option.value, thinkingLevel })}
            />
          ))}
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-xs text-muted-foreground">Thinking</DropdownMenuLabel>
        {THINKING_OPTIONS.map((option) => (
          <ChoiceItem
            key={option.value}
            label={option.label}
            selected={option.value === thinkingLevel}
            onSelect={() => onChange({ model, thinkingLevel: option.value })}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function runnerModelOptions(models: Array<{ id?: string; name?: string }> | undefined) {
  return [
    { value: DRAFT_START_MODEL_AUTO, label: "Auto" },
    ...(models ?? [])
      .filter((item) => (item.id ?? "") !== "")
      .map((item) => ({ value: item.id ?? "", label: item.name || item.id || "" })),
  ];
}

function ChoiceItem({ label, selected, onSelect }: { label: string; selected: boolean; onSelect: () => void }) {
  return (
    <DropdownMenuItem className={MENU_ITEM_CLASS} onSelect={onSelect}>
      <span className="min-w-0 flex-1 break-words">{label}</span>
      {selected ? <Check className="size-3.5" aria-hidden /> : null}
    </DropdownMenuItem>
  );
}
