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
}) {
  const visibleModels = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (needle === "") {
      return modelIds;
    }
    return modelIds.filter((id) => (modelLabels?.[id] ?? id).toLowerCase().includes(needle));
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
        <div className="flex items-center justify-between gap-3">
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
      <div className="max-h-56 space-y-2 overflow-auto">
        {visibleModels.map((model) => (
          <label key={model} className="flex items-center gap-2 text-[13px]">
            <Checkbox
              checked={selected.includes(model)}
              disabled={disabled}
              onChange={(event) => onToggle(model, event.currentTarget.checked)}
            />
            <span className="font-mono text-xs">{modelLabels?.[model] ?? model}</span>
          </label>
        ))}
      </div>
    </div>
  );
}
