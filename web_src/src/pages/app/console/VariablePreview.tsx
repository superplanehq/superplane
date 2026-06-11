import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";

/**
 * Show the resolved value for a variable as a compact field/value list with a
 * one-click "insert" button per field. Falls back to a short message when the
 * lookup found nothing yet (loading) or the source returned no data (error).
 */
export function VariablePreview({
  name,
  value,
  error,
  loading,
  onInsertSnippet,
}: {
  name: string;
  value: unknown;
  error?: string;
  loading: boolean;
  onInsertSnippet: (snippet: string) => void;
}) {
  if (!name.trim()) {
    return <p className="text-[11px] text-slate-400">Give this variable a name to enable the preview.</p>;
  }

  // `useMarkdownVariables` resolves in-flight variables to `null` (not
  // `undefined`) while their backing query loads, so gate on `== null` to cover
  // both. A non-null value during a background refetch still renders its fields
  // (stale-while-revalidate) instead of flashing the loading text.
  if (loading && value == null) {
    return <p className="text-[11px] text-slate-400">Loading preview…</p>;
  }

  if (error) {
    return (
      <p className="text-[11px] text-amber-600" data-testid="markdown-variable-preview-error">
        {error}
      </p>
    );
  }

  if (value == null) {
    return <p className="text-[11px] text-slate-400">No data resolved yet.</p>;
  }

  // List-mode memory variables resolve to an array. Render a dedicated block
  // so authors see it is a list and get insert snippets that use the
  // `join(name.map(...), sep)` pattern instead of a bare `{{ name }}` that
  // would stringify the whole array to JSON.
  if (Array.isArray(value)) {
    return <ListVariablePreviewBlock name={name} items={value} onInsertSnippet={onInsertSnippet} />;
  }

  const fields = previewableFields(value);
  return <VariablePreviewBlock name={name} fields={fields} onInsertSnippet={onInsertSnippet} />;
}

/**
 * Preview block for a list-mode variable. Shows the item count, the fields of
 * the first row (so authors know what each element looks like), and insert
 * snippets that wrap the list in `join(name.map(item, …), ", ")` — the
 * canonical way to render a list since cel-js can't chain `.method()` after a
 * function call and a bare `{{ name }}` would dump raw JSON.
 */
function ListVariablePreviewBlock({
  name,
  items,
  onInsertSnippet,
}: {
  name: string;
  items: unknown[];
  onInsertSnippet: (snippet: string) => void;
}) {
  const [collapsed, setCollapsed] = useState(false);
  const fieldKeys = listItemFieldKeys(items);
  const countSnippet = `{{ size(${name}) }}`;

  return (
    <div className="min-w-0 space-y-1 rounded border border-slate-100 bg-slate-50 p-2">
      <div className="flex min-w-0 items-center gap-2 text-[11px] text-slate-500">
        <button
          type="button"
          onClick={() => setCollapsed((prev) => !prev)}
          aria-expanded={!collapsed}
          className="flex shrink-0 items-center gap-1 rounded px-1 py-0.5 hover:bg-slate-200/60"
          data-testid="markdown-variable-preview-toggle"
        >
          {collapsed ? <ChevronRight className="size-3" /> : <ChevronDown className="size-3" />}
          <span className="font-semibold uppercase tracking-wide">Preview</span>
        </button>
        <span className="rounded bg-slate-200/70 px-1 text-[10px] font-medium text-slate-600">
          List · {items.length} {items.length === 1 ? "item" : "items"}
        </span>
        <span className="min-w-0 flex-1" />
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-5 shrink-0 px-1.5 text-[11px]"
          onClick={() => onInsertSnippet(countSnippet)}
          title={`Insert ${countSnippet}`}
        >
          Count
        </Button>
      </div>
      {!collapsed ? (
        <ListVariablePreviewBody name={name} items={items} fieldKeys={fieldKeys} onInsertSnippet={onInsertSnippet} />
      ) : null}
    </div>
  );
}

function ListVariablePreviewBody({
  name,
  items,
  fieldKeys,
  onInsertSnippet,
}: {
  name: string;
  items: unknown[];
  fieldKeys: string[];
  onInsertSnippet: (snippet: string) => void;
}) {
  if (items.length === 0) {
    return <p className="text-[11px] text-slate-500">No rows matched yet. The list resolves to an empty array.</p>;
  }
  // Scalar lists (strings, numbers) have no fields to map over, so offer a
  // direct join of the whole list.
  if (fieldKeys.length === 0) {
    const joinSnippet = `{{ join(${name}, ", ") }}`;
    return (
      <div className="space-y-1">
        <p className="text-[11px] text-slate-500" title={shortPreviewString(items[0])}>
          e.g. {shortPreviewString(items[0])}
        </p>
        <button
          type="button"
          onClick={() => onInsertSnippet(joinSnippet)}
          className="truncate text-left font-mono text-[11px] text-sky-700 underline-offset-2 hover:underline"
          title={`Insert ${joinSnippet}`}
        >
          {joinSnippet}
        </button>
      </div>
    );
  }
  const first = items[0] as Record<string, unknown>;
  return (
    <ul className="max-h-40 space-y-0.5 overflow-y-auto pr-1">
      {fieldKeys.map((key) => {
        const snippet = `{{ join(${name}.map(item, item${memberAccessor(key)}), ", ") }}`;
        return (
          <li key={key} className="flex min-w-0 items-start gap-2">
            <button
              type="button"
              onClick={() => onInsertSnippet(snippet)}
              className="max-w-[55%] shrink-0 truncate text-left font-mono text-[11px] text-sky-700 underline-offset-2 hover:underline"
              title={`Insert ${snippet}`}
            >
              {key}
            </button>
            <span className="min-w-0 flex-1 truncate text-[11px] text-slate-600" title={shortPreviewString(first[key])}>
              {shortPreviewString(first[key])}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

/**
 * Field keys to offer for a list variable: the own keys of the first
 * object-shaped element, minus internal CEL aliases. Returns `[]` for scalar
 * lists (or an empty list) so the caller falls back to a whole-list `join`.
 */
function listItemFieldKeys(items: unknown[]): string[] {
  const first = items[0];
  if (!first || typeof first !== "object" || Array.isArray(first)) return [];
  return Object.keys(first as Record<string, unknown>)
    .filter((key) => !INTERNAL_PREVIEW_KEYS.has(key) && key !== "$")
    .slice(0, 12);
}

/**
 * Inner collapsible body for `VariablePreview`. Pulled out so the collapsed
 * state hook can live on a real component instance (one per variable card)
 * without forcing the outer wrapper through all of its early-return cases.
 *
 * The card itself never widens: long preview values are clipped with
 * `truncate` and rendered inside a `min-w-0 flex-1` cell so flex's default
 * content-min-width doesn't push the card past the variables-panel column.
 */
function VariablePreviewBlock({
  name,
  fields,
  onInsertSnippet,
}: {
  name: string;
  fields: Array<{ key: string; preview: string; accessor: string }>;
  onInsertSnippet: (snippet: string) => void;
}) {
  const insertable = (accessor: string) => `{{ ${name}${accessor} }}`;
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="min-w-0 space-y-1 rounded border border-slate-100 bg-slate-50 p-2">
      <div className="flex min-w-0 items-center gap-2 text-[11px] text-slate-500">
        <button
          type="button"
          onClick={() => setCollapsed((prev) => !prev)}
          aria-expanded={!collapsed}
          className="flex shrink-0 items-center gap-1 rounded px-1 py-0.5 hover:bg-slate-200/60"
          data-testid="markdown-variable-preview-toggle"
        >
          {collapsed ? <ChevronRight className="size-3" /> : <ChevronDown className="size-3" />}
          <span className="font-semibold uppercase tracking-wide">Preview</span>
        </button>
        <span className="min-w-0 flex-1" />
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-5 shrink-0 px-1.5 text-[11px]"
          onClick={() => onInsertSnippet(insertable(""))}
          title={`Insert ${insertable("")}`}
        >
          Insert
        </Button>
      </div>
      {!collapsed ? (
        fields.length === 0 ? (
          <p className="text-[11px] text-slate-500">Resolved value has no readable fields.</p>
        ) : (
          // The list can grow taller than the card on rich objects (memory
          // rows, runs with many `$["Node"]` accessors). Cap its height so
          // the variable card stays compact and let the list scroll
          // vertically inside the preview without affecting the outer
          // variables-panel scroll.
          <ul className="max-h-40 space-y-0.5 overflow-y-auto pr-1">
            {fields.map((field) => (
              <li key={field.key} className="flex min-w-0 items-start gap-2">
                <button
                  type="button"
                  onClick={() => onInsertSnippet(insertable(field.accessor))}
                  className="max-w-[55%] shrink-0 truncate text-left font-mono text-[11px] text-sky-700 underline-offset-2 hover:underline"
                  title={`Insert ${insertable(field.accessor)}`}
                >
                  {field.key}
                </button>
                <span className="min-w-0 flex-1 truncate text-[11px] text-slate-600" title={field.preview}>
                  {field.preview}
                </span>
              </li>
            ))}
          </ul>
        )
      ) : null}
    </div>
  );
}

/**
 * Pick a short list of (key, preview-string) pairs from a resolved variable
 * value. Object-shaped values (memory rows, run rows) get a flat list of
 * their own top-level keys. For runs we surface the `$["Node"]` accessors so
 * authors can see which node references are available. Skips internal keys
 * (the CEL rewrite alias and the `$` map itself, which we expose via derived
 * rows) to keep the list focused on user-visible fields.
 */
const INTERNAL_PREVIEW_KEYS = new Set(["__runNodes__"]);

/**
 * Matches a key that is a bare CEL identifier (`status`, `_id`, `nodeName`).
 * Anything else — dashes, dots, spaces, leading digits — must be reached with
 * bracket access so the generated snippet stays valid and targets the right
 * path.
 */
const IDENTIFIER_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

/**
 * Build the member-access suffix that follows a base expression for a single
 * field key. Identifier-safe keys use dot access (`.status`); everything else
 * falls back to quoted bracket access (`["deploy-status"]`) so keys with
 * dashes, dots, spaces, or leading digits don't emit invalid CEL or silently
 * resolve the wrong path.
 */
function memberAccessor(key: string): string {
  return IDENTIFIER_RE.test(key) ? `.${key}` : `[${JSON.stringify(key)}]`;
}

function previewableFields(value: unknown): Array<{ key: string; preview: string; accessor: string }> {
  if (!value || typeof value !== "object") {
    return [{ key: "", preview: shortPreviewString(value), accessor: "" }];
  }
  const record = value as Record<string, unknown>;
  const out: Array<{ key: string; preview: string; accessor: string }> = [];
  for (const key of Object.keys(record)) {
    if (INTERNAL_PREVIEW_KEYS.has(key)) continue;
    if (key === "$") {
      const nodes = record.$ as Record<string, unknown> | undefined;
      if (nodes && typeof nodes === "object") {
        for (const nodeKey of Object.keys(nodes)) {
          const display = `$[${JSON.stringify(nodeKey)}].data`;
          out.push({
            key: display,
            preview: shortPreviewString((nodes[nodeKey] as { data?: unknown })?.data),
            accessor: `.${display}`,
          });
        }
      }
      continue;
    }
    out.push({ key, preview: shortPreviewString(record[key]), accessor: memberAccessor(key) });
  }
  return out.slice(0, 12);
}

function shortPreviewString(value: unknown): string {
  if (value === null || value === undefined) return "-";
  if (typeof value === "string") return value.length > 80 ? `${value.slice(0, 80)}…` : value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    const json = JSON.stringify(value);
    return json.length > 80 ? `${json.slice(0, 80)}…` : json;
  } catch {
    return "[unserializable]";
  }
}
