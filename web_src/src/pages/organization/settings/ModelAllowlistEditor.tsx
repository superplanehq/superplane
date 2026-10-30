import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Search } from "lucide-react";
import { useMemo } from "react";

export function ModelAllowlistEditor({
  modelIds,
  selected,
  query,
  onQueryChange,
  onToggle,
  onBulkToggle,
  disabled,
  searchLabel,
  showCount = false,
  showBulkToggle = false,
  modelLabels,
  unavailableModels = [],
  unavailableLabel = "Unavailable",
}: {
  modelIds: string[];
  selected: string[];
  query: string;
  onQueryChange: (query: string) => void;
  onToggle: (model: string, checked: boolean) => void;
  onBulkToggle?: () => void;
  disabled: boolean;
  searchLabel: string;
  showCount?: boolean;
  showBulkToggle?: boolean;
  modelLabels?: Record<string, string>;
  unavailableModels?: string[];
  unavailableLabel?: string;
}) {
  const visibleModels = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (needle === "") {
      return modelIds;
    }
    return modelIds.filter((id) => `${modelLabels?.[id] ?? ""} ${id}`.toLowerCase().includes(needle));
  }, [modelIds, modelLabels, query]);
  const allSelected = modelIds.length > 0 && modelIds.every((model) => selected.includes(model));

  return (
    <div className="space-y-3">
      <InputGroup>
        <InputGroupAddon>
          <Search className="size-3.5" />
        </InputGroupAddon>
        <InputGroupInput
          type="search"
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Search models..."
          aria-label={searchLabel}
        />
      </InputGroup>
      {showCount ? (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            {selected.length} of {modelIds.length} models selected
          </p>
          {showBulkToggle && onBulkToggle ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-auto shrink-0 px-2 py-1 text-xs"
              disabled={disabled || modelIds.length === 0}
              onClick={onBulkToggle}
              data-testid="model-allowlist-bulk-toggle"
            >
              {allSelected ? "Deselect all" : "Select all"}
            </Button>
          ) : null}
        </div>
      ) : null}
      <div
        role="region"
        aria-label="Model choices"
        tabIndex={0}
        className="max-h-56 space-y-2 overflow-auto focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        {visibleModels.map((model) => (
          <label key={model} className="flex min-w-0 items-start gap-2 rounded-md px-1 py-1.5 text-[13px]">
            <Checkbox
              className="mt-0.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              checked={selected.includes(model)}
              disabled={disabled}
              onChange={(event) => onToggle(model, event.currentTarget.checked)}
            />
            <span className="min-w-0 break-words [overflow-wrap:anywhere]">
              <span className="block">{modelLabels?.[model] ?? model}</span>
              {unavailableModels.includes(model) ? (
                <span className="block text-xs text-muted-foreground">{unavailableLabel}</span>
              ) : null}
            </span>
          </label>
        ))}
        {visibleModels.length === 0 ? (
          <p role="status" className="py-2 text-xs text-muted-foreground">
            {modelIds.length === 0 ? "No models are available." : "No matching models. Try a different search."}
          </p>
        ) : null}
      </div>
    </div>
  );
}
