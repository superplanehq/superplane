import { Checkbox } from "@/components/ui/checkbox";
import { Input, InputGroup } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { filterModelIds } from "@/lib/hostedLLMModels";
import { Search } from "lucide-react";
import { useMemo } from "react";

export function ModelAllowlistEditor({
  modelIds,
  selected,
  query,
  onQueryChange,
  onToggle,
  disabled,
  searchLabel,
  showCount = false,
}: {
  modelIds: string[];
  selected: string[];
  query: string;
  onQueryChange: (query: string) => void;
  onToggle: (model: string, checked: boolean) => void;
  disabled: boolean;
  searchLabel: string;
  showCount?: boolean;
}) {
  const visibleModels = useMemo(() => filterModelIds(modelIds, query), [modelIds, query]);

  return (
    <div className="space-y-3">
      <InputGroup className="relative">
        <Search
          size={14}
          className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          type="search"
          className="pl-9"
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Search models..."
          aria-label={searchLabel}
        />
      </InputGroup>
      {showCount ? (
        <Text className="text-xs text-muted-foreground">
          {selected.length} of {modelIds.length} models selected
        </Text>
      ) : null}
      <div className="max-h-56 space-y-2 overflow-auto">
        {visibleModels.map((model) => (
          <label key={model} className="flex items-center gap-2 text-[13px]">
            <Checkbox
              checked={selected.includes(model)}
              disabled={disabled}
              onChange={(event) => onToggle(model, event.currentTarget.checked)}
            />
            <span className="font-mono text-xs">{model}</span>
          </label>
        ))}
      </div>
    </div>
  );
}
