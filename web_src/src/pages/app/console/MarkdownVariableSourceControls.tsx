import { useMemo } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { statusesCompatibleWithRunSelect } from "@/ui/Runs/runStatusTriggerFilter";

import {
  MARKDOWN_RUN_SELECTS,
  MARKDOWN_VARIABLE_DIRECTIONS,
  type MarkdownMemoryVariableSource,
  type MarkdownRunSelect,
  type MarkdownRunVariableSource,
  type MarkdownVariableDirection,
  type MarkdownVariableMatch,
  type MarkdownVariableMode,
} from "./panelTypes";
import { RunDataSourceFiltersPanel } from "./RunDataSourceFiltersPanel";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

const RUN_SELECT_LABELS: Record<MarkdownRunSelect, string> = {
  latest: "Latest run",
  latest_passed: "Latest passed run",
  latest_failed: "Latest failed run",
};

const DIRECTION_LABELS: Record<MarkdownVariableDirection, string> = {
  desc: "Descending",
  asc: "Ascending",
};

const RESULT_MODE_LABELS: Record<MarkdownVariableMode, string> = {
  single: "Single row",
  list: "List of rows",
};

export function MemorySourceControls({
  canvasId,
  source,
  onChange,
}: {
  canvasId: string;
  source: MarkdownMemoryVariableSource;
  onChange: (next: MarkdownMemoryVariableSource) => void;
}) {
  const { namespaces, fields } = useMemoryCatalog(canvasId, source.namespace);
  const orderByOptions = useMemo(() => {
    const set = new Set<string>(["createdAt", "updatedAt"]);
    for (const field of fields) set.add(field.field);
    if (source.orderBy?.trim()) set.add(source.orderBy.trim());
    return Array.from(set);
  }, [fields, source.orderBy]);

  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <Label className="text-[11px] font-medium text-slate-600">Namespace</Label>
        <Select
          value={source.namespace || undefined}
          onValueChange={(value) => onChange({ ...source, namespace: value })}
        >
          <SelectTrigger className="h-7 text-[12px]" data-testid="markdown-variable-memory-namespace">
            <SelectValue placeholder="Select a namespace" />
          </SelectTrigger>
          <SelectContent>
            {namespaces.length === 0 ? (
              <SelectItem value="__empty__" disabled>
                No namespaces in this canvas
              </SelectItem>
            ) : (
              namespaces.map((ns) => (
                <SelectItem key={ns.namespace} value={ns.namespace}>
                  {ns.namespace}
                  <span className="ml-1 text-[11px] text-slate-400">({ns.count})</span>
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label className="text-[11px] font-medium text-slate-600">Order by</Label>
          <Select
            value={source.orderBy || "createdAt"}
            onValueChange={(value) => onChange({ ...source, orderBy: value })}
          >
            <SelectTrigger className="h-7 text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {orderByOptions.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-[11px] font-medium text-slate-600">Direction</Label>
          <Select
            value={source.direction ?? "desc"}
            onValueChange={(value) => onChange({ ...source, direction: value as MarkdownVariableDirection })}
          >
            <SelectTrigger className="h-7 text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MARKDOWN_VARIABLE_DIRECTIONS.map((dir) => (
                <SelectItem key={dir} value={dir}>
                  {DIRECTION_LABELS[dir]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <MemoryMatchesEditor matches={source.matches ?? []} onChange={(next) => onChange({ ...source, matches: next })} />

      <MemoryResultControls source={source} onChange={onChange} />
    </div>
  );
}

function MemoryResultControls({
  source,
  onChange,
}: {
  source: MarkdownMemoryVariableSource;
  onChange: (next: MarkdownMemoryVariableSource) => void;
}) {
  const mode: MarkdownVariableMode = source.mode === "list" ? "list" : "single";

  const handleModeChange = (next: MarkdownVariableMode) => {
    if (next === "single") {
      // Strip `mode` / `limit` so single-row variables keep the minimal YAML
      // they had before list mode existed.
      const { mode: _mode, limit: _limit, ...rest } = source;
      void _mode;
      void _limit;
      onChange(rest);
      return;
    }
    onChange({ ...source, mode: "list" });
  };

  const handleLimitChange = (raw: string) => {
    const trimmed = raw.trim();
    if (trimmed === "") {
      const { limit: _limit, ...rest } = source;
      void _limit;
      onChange(rest);
      return;
    }
    const parsed = Number(trimmed);
    if (!Number.isFinite(parsed) || !Number.isInteger(parsed) || parsed <= 0) {
      // Keep what the author is typing locally; persisted state stays clean
      // until the value parses as a positive integer. The validator will
      // surface a precise error on save if they try to commit garbage.
      return;
    }
    onChange({ ...source, mode: "list", limit: parsed });
  };

  return (
    <div className="space-y-1">
      <Label className="text-[11px] font-medium text-slate-600">Result</Label>
      <Select value={mode} onValueChange={(value) => handleModeChange(value as MarkdownVariableMode)}>
        <SelectTrigger className="h-7 text-[12px]" data-testid="markdown-variable-memory-mode">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="single">{RESULT_MODE_LABELS.single}</SelectItem>
          <SelectItem value="list">{RESULT_MODE_LABELS.list}</SelectItem>
        </SelectContent>
      </Select>
      {mode === "list" ? (
        <div className="space-y-1 pt-1">
          <Label className="text-[11px] font-medium text-slate-600">Limit</Label>
          <Input
            type="number"
            inputMode="numeric"
            min={1}
            step={1}
            value={source.limit ?? ""}
            onChange={(e) => handleLimitChange(e.target.value)}
            placeholder="All matching rows"
            className="h-7 text-[12px]"
            aria-label="Limit list to N rows"
            data-testid="markdown-variable-memory-limit"
          />
          <p className="text-[11px] text-slate-400">
            Leave empty to include every match. Use CEL list ops like{" "}
            <code className="rounded bg-slate-100 px-1 text-[10px]">name.map(r, r.field)</code> in {"{{ }}"}.
          </p>
        </div>
      ) : null}
    </div>
  );
}

function MemoryMatchesEditor({
  matches,
  onChange,
}: {
  matches: MarkdownVariableMatch[];
  onChange: (next: MarkdownVariableMatch[]) => void;
}) {
  const updateMatch = (index: number, next: MarkdownVariableMatch) => {
    const out = matches.slice();
    out[index] = next;
    onChange(out);
  };
  const addMatch = () => onChange([...matches, { field: "", value: "" }]);
  const removeMatch = (index: number) => onChange(matches.filter((_, i) => i !== index));

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <Label className="text-[11px] font-medium text-slate-600">Match (field equals value)</Label>
        <Button type="button" size="sm" variant="outline" className="h-6 gap-1" onClick={addMatch}>
          <Plus className="size-3" />
          Add match
        </Button>
      </div>
      {matches.length === 0 ? (
        <p className="text-[11px] text-slate-400">Optional. Add a match to pick a specific row.</p>
      ) : (
        <div className="space-y-1.5">
          {matches.map((match, index) => (
            <div key={index} className="flex min-w-0 items-center gap-1">
              <Input
                value={match.field}
                onChange={(e) => updateMatch(index, { ...match, field: e.target.value })}
                placeholder="field"
                className="h-7 min-w-0 flex-1 text-[12px]"
                aria-label="Match field"
              />
              <Input
                value={match.value}
                onChange={(e) => updateMatch(index, { ...match, value: e.target.value })}
                placeholder="value"
                className="h-7 min-w-0 flex-1 text-[12px]"
                aria-label="Match value"
              />
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={() => removeMatch(index)}
                aria-label="Remove match"
                className="h-7 w-7 shrink-0 text-slate-500 hover:bg-red-50 hover:text-red-600"
              >
                <Trash2 className="size-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function RunSourceControls({
  source,
  onChange,
}: {
  source: MarkdownRunVariableSource;
  onChange: (next: MarkdownRunVariableSource) => void;
}) {
  const setStatuses = (statuses: RunStatusFilter[] | undefined) => {
    const next = { ...source };
    const compatibleStatuses = statusesCompatibleWithRunSelect(source.select, statuses);
    if (compatibleStatuses) next.statuses = compatibleStatuses;
    else delete next.statuses;
    onChange(next);
  };
  const setTriggers = (triggers: string[] | undefined) => {
    const next = { ...source };
    if (triggers && triggers.length > 0) next.triggers = triggers;
    else delete next.triggers;
    onChange(next);
  };
  const setSelect = (select: MarkdownRunSelect) => {
    const next: MarkdownRunVariableSource = { ...source, select };
    const statuses = statusesCompatibleWithRunSelect(select, source.statuses);
    if (statuses && statuses.length > 0) next.statuses = statuses;
    else delete next.statuses;
    onChange(next);
  };

  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <Label className="text-[11px] font-medium text-slate-600">Run</Label>
        <Select value={source.select} onValueChange={(value) => setSelect(value as MarkdownRunSelect)}>
          <SelectTrigger className="h-7 text-[12px]" data-testid="markdown-variable-run-select">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {MARKDOWN_RUN_SELECTS.map((select) => (
              <SelectItem key={select} value={select}>
                {RUN_SELECT_LABELS[select]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <RunDataSourceFiltersPanel
        statuses={source.statuses}
        triggers={source.triggers}
        onStatusesChange={setStatuses}
        onTriggersChange={setTriggers}
        testIdSuffix="markdown-variable"
      />
    </div>
  );
}
