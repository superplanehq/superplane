import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { cn } from "@/lib/utils";
import type { SelectableLLMModel } from "@/lib/selectableLLMModels";
import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";

export type CreateWithAgentModelPickerProps = {
  models: SelectableLLMModel[];
  selectedKey: string;
  selectedLabel: string;
  disabled?: boolean;
  loading?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onSelect: (key: string) => void;
};

export function CreateWithAgentModelPicker({
  models,
  selectedKey,
  selectedLabel,
  disabled = false,
  loading = false,
  open,
  onOpenChange,
  onSelect,
}: CreateWithAgentModelPickerProps) {
  const label = selectedLabel.trim() || CREATE_WITH_AGENT_COPY.selectModel;
  const grouped = groupSelectableModels(models);
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const pickerOpen = open ?? uncontrolledOpen;
  const setPickerOpen = onOpenChange ?? setUncontrolledOpen;

  return (
    <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "max-w-[16rem] truncate text-[12px] text-muted-foreground underline decoration-wavy decoration-muted-foreground/70 underline-offset-4",
            "hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50 disabled:no-underline",
          )}
          disabled={disabled || loading}
          data-testid="create-with-agent-model"
          aria-label={CREATE_WITH_AGENT_COPY.usingModel(label)}
        >
          {CREATE_WITH_AGENT_COPY.usingModel(label)}
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-1" data-testid="create-with-agent-model-picker">
        {models.length === 0 ? (
          <p className="px-2 py-1.5 text-[12px] text-muted-foreground">{CREATE_WITH_AGENT_COPY.noModels}</p>
        ) : (
          grouped.map((group) => (
            <div key={group.id} className="py-1">
              <p className="px-2 py-1 text-[11px] font-medium text-muted-foreground">{group.name}</p>
              {group.models.map((model) => (
                <button
                  key={model.key}
                  type="button"
                  className={cn(
                    "flex w-full flex-col items-start rounded-md px-2 py-1.5 text-left text-[12px] hover:bg-muted",
                    model.key === selectedKey ? "bg-muted" : "",
                  )}
                  data-testid={`create-with-agent-model-option-${model.key}`}
                  onClick={() => {
                    onSelect(model.key);
                    setPickerOpen(false);
                  }}
                >
                  <span className="text-foreground">{model.label}</span>
                  <span className="text-[11px] text-muted-foreground">{model.source.name}</span>
                </button>
              ))}
            </div>
          ))
        )}
      </PopoverContent>
    </Popover>
  );
}

function groupSelectableModels(models: SelectableLLMModel[]): Array<{
  id: string;
  name: string;
  models: SelectableLLMModel[];
}> {
  const groups: Array<{ id: string; name: string; models: SelectableLLMModel[] }> = [];
  for (const model of models) {
    const existing = groups.find((group) => group.id === model.source.id);
    if (existing) {
      existing.models.push(model);
      continue;
    }
    groups.push({ id: model.source.id, name: model.source.name, models: [model] });
  }
  return groups;
}
