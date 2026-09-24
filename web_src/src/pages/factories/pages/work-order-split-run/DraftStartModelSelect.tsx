import { Bot, Check, ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

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

const MENU_LABEL_CLASSNAME = "text-[11px] font-medium tracking-[0.04em] text-muted-foreground";
const MENU_ITEM_CLASSNAME = "cursor-pointer text-[13px]";

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
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Model</DropdownMenuLabel>
        <DropdownMenuItem
          className={MENU_ITEM_CLASSNAME}
          onSelect={() => onChange({ model: DRAFT_START_MODEL_AUTO, thinkingLevel })}
        >
          <span className="flex-1">Auto</span>
          {model === DRAFT_START_MODEL_AUTO ? <Check className="size-3.5" aria-hidden /> : null}
        </DropdownMenuItem>
        {(models.data ?? []).map((item) => {
          const id = item.id ?? "";
          if (id === "") {
            return null;
          }
          return (
            <DropdownMenuItem
              key={id}
              className={MENU_ITEM_CLASSNAME}
              onSelect={() => onChange({ model: id, thinkingLevel })}
            >
              <span className="flex-1">{item.name || id}</span>
              {model === id ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}
        <DropdownMenuSeparator />
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Thinking</DropdownMenuLabel>
        {START_THINKING_LEVELS.map((level) => (
          <DropdownMenuItem
            key={level.value}
            className={MENU_ITEM_CLASSNAME}
            onSelect={() => onChange({ model, thinkingLevel: level.value })}
          >
            <span className="flex-1">{level.label}</span>
            {thinkingLevel === level.value ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
