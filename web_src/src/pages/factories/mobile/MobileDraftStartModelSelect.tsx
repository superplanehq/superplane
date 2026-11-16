import { Check, ChevronDown } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import { useFactoryLineRunnerModels } from "@/hooks/useFactoryLineRunnerModels";
import { THINKING_LEVEL_HIGH, THINKING_LEVEL_LOW, THINKING_LEVEL_MEDIUM } from "@/lib/thinkingLevel";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import {
  draftStartDefaultLabels,
  draftStartModelOptions,
  resolveDraftStartSelection,
  type DraftStartSelection,
} from "../pages/work-order-split-run/draftStartModel";
import { draftStartDefaultsStatus, type DraftStartDefaults } from "@/hooks/useDraftStartDefaults";
import { DraftStartSelectionLabel } from "../pages/work-order-split-run/DraftStartSelectionLabel";

const THINKING_OPTIONS = [
  { value: THINKING_LEVEL_LOW, label: "Low" },
  { value: THINKING_LEVEL_MEDIUM, label: "Medium" },
  { value: THINKING_LEVEL_HIGH, label: "High" },
];

const MENU_ITEM_CLASS = "cursor-pointer text-[13px]";

export function MobileDraftStartModelSelect({
  organizationId,
  factoryId,
  lineName,
  orderId,
  model,
  thinkingLevel,
  defaults,
  onChange,
  disabled = false,
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
}) {
  const models = useFactoryLineRunnerModels(organizationId, factoryId, lineName, Boolean(lineName), orderId);
  const catalog = models.data?.models;
  const modelOptions = useMemo(() => draftStartModelOptions(catalog), [catalog]);
  const defaultLabels = draftStartDefaultLabels(defaults.values, modelOptions);
  const resolved = resolveDraftStartSelection(
    { model, thinkingLevel },
    defaults.values ?? {},
    modelOptions,
    defaults.values ? undefined : draftStartDefaultsStatus(defaults),
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="outline"
          aria-label={resolved.accessibleLabel}
          data-testid="split-run-draft-model"
          disabled={disabled}
          className="h-7 max-w-36 min-w-0 shrink gap-1.5 overflow-hidden !rounded-none border-0 bg-background px-2 text-xs shadow-none"
        >
          <DraftStartSelectionLabel resolved={resolved} thinkingLevel={thinkingLevel} />
          <ChevronDown className="size-3 shrink-0 opacity-60" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        side="bottom"
        collisionPadding={8}
        className="w-[min(18rem,var(--radix-dropdown-menu-content-available-width))] max-w-[calc(100vw-1rem)]"
      >
        {!defaults.values ? (
          <DropdownMenuItem onSelect={defaults.retry}>Retry Implementation defaults</DropdownMenuItem>
        ) : null}
        <DropdownMenuLabel className="break-words text-xs text-muted-foreground">
          Model — {resolved.modelLabel}
        </DropdownMenuLabel>
        <div className="max-h-52 overflow-y-auto" data-testid="mobile-task-model-list">
          {[{ value: "", label: defaultLabels.model }, ...(lineName ? modelOptions : [])].map((option) => (
            <ChoiceItem
              key={option.value}
              label={option.label}
              selected={option.value === (model ?? "")}
              onSelect={() => onChange({ model: option.value || undefined, thinkingLevel })}
            />
          ))}
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="break-words text-xs text-muted-foreground">
          Thinking — {resolved.thinkingLabel}
        </DropdownMenuLabel>
        {[{ value: "", label: defaultLabels.thinking }, ...THINKING_OPTIONS].map((option) => (
          <ChoiceItem
            key={option.value}
            label={option.label}
            selected={option.value === (thinkingLevel ?? "")}
            onSelect={() => onChange({ model, thinkingLevel: option.value || undefined })}
          />
        ))}
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

function ChoiceItem({ label, selected, onSelect }: { label: string; selected: boolean; onSelect: () => void }) {
  return (
    <DropdownMenuItem className={MENU_ITEM_CLASS} onSelect={onSelect}>
      <span className="min-w-0 flex-1 break-words">{label}</span>
      {selected ? <Check className="size-3.5" aria-hidden /> : null}
    </DropdownMenuItem>
  );
}
