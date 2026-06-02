import { Button } from "@/components/ui/button";
import type { CanvasMemoryEntry, CanvasMemoryEntrySource } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronDown, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { CanvasMemoryBankDialog, type CanvasMemoryBankDialogMode } from "./CanvasMemoryBankDialog";

export type CanvasMemoryViewProps = {
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  canEdit?: boolean;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
  onCreateBank?: (input: { namespace: string; entries: unknown[] }) => Promise<void>;
  isCreatingBank?: boolean;
  onUpdateBank?: (input: { namespace: string; newNamespace?: string; entries: unknown[] }) => Promise<void>;
  isUpdatingBank?: boolean;
};

interface BankGroup {
  namespace: string;
  source: CanvasMemoryEntrySource;
  entries: CanvasMemoryEntry[];
}

function groupBanks(entries: CanvasMemoryEntry[]): BankGroup[] {
  const groups = new Map<string, BankGroup>();
  for (const entry of entries) {
    const namespace = entry.namespace || "(no namespace)";
    const existing = groups.get(namespace);
    if (existing) {
      existing.entries.push(entry);
      continue;
    }
    groups.set(namespace, {
      namespace,
      source: entry.source,
      entries: [entry],
    });
  }
  return Array.from(groups.values());
}

export function CanvasMemoryView(props: CanvasMemoryViewProps) {
  const leftOffset = useEffectiveLeftSidebarWidth();

  return (
    <div
      className="absolute bottom-0 top-[5rem] z-10 flex flex-col overflow-hidden bg-slate-50"
      style={{ left: leftOffset, right: 0 }}
      data-testid="memory-overlay"
    >
      <CanvasMemoryViewBody {...props} />
    </div>
  );
}

type DialogState =
  | { open: false }
  | { open: true; mode: "create" }
  | { open: true; mode: "edit"; namespace: string; entries: unknown[] };

function computeIsSubmitting(
  dialogState: DialogState,
  isCreatingBank: boolean | undefined,
  isUpdatingBank: boolean | undefined,
): boolean {
  if (!dialogState.open) {
    return false;
  }
  if (dialogState.mode === "create") {
    return !!isCreatingBank;
  }
  return !!isUpdatingBank;
}

function CanvasMemoryViewBody({
  entries,
  isLoading,
  errorMessage,
  canEdit,
  onDeleteEntry,
  deletingId,
  onCreateBank,
  isCreatingBank,
  onUpdateBank,
  isUpdatingBank,
}: CanvasMemoryViewProps) {
  const banks = useMemo(() => groupBanks(entries), [entries]);
  const [dialogState, setDialogState] = useState<DialogState>({ open: false });

  const closeDialog = () => setDialogState({ open: false });

  const handleCreateBankClick = () => {
    setDialogState({ open: true, mode: "create" });
  };

  const handleEditBankClick = (bank: BankGroup) => {
    setDialogState({
      open: true,
      mode: "edit",
      namespace: bank.namespace,
      entries: bank.entries.map((entry) => entry.values),
    });
  };

  const handleDialogSubmit = async (input: { namespace: string; entries: unknown[] }) => {
    if (!dialogState.open) return;
    if (dialogState.mode === "create") {
      if (!onCreateBank) return;
      await onCreateBank(input);
      return;
    }
    if (!onUpdateBank) return;
    await onUpdateBank({
      namespace: dialogState.namespace,
      newNamespace: input.namespace !== dialogState.namespace ? input.namespace : undefined,
      entries: input.entries,
    });
  };

  const showCreateButton = !!canEdit && !!onCreateBank;
  const isSubmitting = computeIsSubmitting(dialogState, isCreatingBank, isUpdatingBank);

  const dialogMode: CanvasMemoryBankDialogMode | undefined = dialogState.open ? dialogState.mode : undefined;
  const dialogNamespace = dialogState.open && dialogState.mode === "edit" ? dialogState.namespace : undefined;
  const dialogInitialEntries = dialogState.open && dialogState.mode === "edit" ? dialogState.entries : undefined;

  if (isLoading) {
    return (
      <div className="flex min-h-0 w-full flex-1 items-center justify-center px-4 py-12 text-[13px] text-gray-500">
        Loading memory entries…
      </div>
    );
  }

  if (errorMessage) {
    return (
      <div className="flex min-h-0 w-full flex-1 flex-col items-center justify-center gap-2 px-6 py-12 text-center">
        <p className="text-[13px] font-medium text-red-600">Failed to load memory entries.</p>
        <p className="max-w-md text-xs text-red-500">{errorMessage}</p>
      </div>
    );
  }

  return (
    <>
      {showCreateButton ? (
        <div className="flex items-center justify-end gap-2 border-b border-slate-950/10 bg-white px-4 py-2">
          <Button type="button" size="sm" onClick={handleCreateBankClick} data-testid="memory-create-bank-button">
            <Plus className="h-4 w-4" aria-hidden="true" />
            Create memory bank
          </Button>
        </div>
      ) : null}
      {banks.length === 0 ? (
        <ZeroState canCreate={showCreateButton} onCreate={handleCreateBankClick} />
      ) : (
        <div className="min-h-0 w-full min-w-0 flex-1 overflow-auto">
          {banks.map((bank) => (
            <NamespaceSection
              key={bank.namespace}
              bank={bank}
              canEdit={!!canEdit}
              onDeleteEntry={onDeleteEntry}
              deletingId={deletingId}
              onEditBank={onUpdateBank ? () => handleEditBankClick(bank) : undefined}
            />
          ))}
        </div>
      )}
      {dialogMode ? (
        <CanvasMemoryBankDialog
          open={dialogState.open}
          onOpenChange={(open) => {
            if (!open) closeDialog();
          }}
          mode={dialogMode}
          originalNamespace={dialogNamespace}
          initialEntries={dialogInitialEntries}
          isSubmitting={isSubmitting}
          onSubmit={handleDialogSubmit}
        />
      ) : null}
    </>
  );
}

type NamespaceSectionProps = {
  bank: BankGroup;
  canEdit: boolean;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
  onEditBank?: () => void;
};

function NamespaceSection({ bank, canEdit, onDeleteEntry, deletingId, onEditBank }: NamespaceSectionProps) {
  const [isOpen, setIsOpen] = useState(true);
  const { namespace, source, entries } = bank;
  const isManual = source === "manual";
  const showEdit = canEdit && isManual && !!onEditBank;

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className="group/section m-4 overflow-hidden rounded-md border border-slate-950/15 bg-white"
      data-testid={`memory-namespace-section-${namespace}`}
    >
      <div className="flex w-full items-center gap-2 border-b border-slate-950/15 px-3 py-2 text-left font-mono text-[13px] text-gray-600 group-data-[state=closed]/section:border-b-0">
        <CollapsibleTrigger
          className="group flex flex-1 items-center gap-2 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          data-testid={`memory-namespace-toggle-${namespace}`}
          aria-label={`Toggle ${namespace} namespace`}
        >
          <ChevronDown
            aria-hidden="true"
            className="size-4 shrink-0 text-gray-500 transition-transform duration-150 group-data-[state=closed]:-rotate-90"
          />
          <span className="flex-1 truncate text-left">Namespace: {namespace}</span>
        </CollapsibleTrigger>
        <SourceBadge source={source} />
        <span className="shrink-0 font-sans text-[13px] font-medium text-gray-500">
          {entries.length} {entries.length === 1 ? "item" : "items"}
        </span>
        {showEdit ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onEditBank}
            className="text-gray-500 hover:text-gray-900"
            title="Edit memory bank"
            data-testid={`memory-namespace-edit-${namespace}`}
          >
            <Pencil className="h-4 w-4" />
          </Button>
        ) : null}
      </div>
      <CollapsibleContent>{renderNamespaceTable(entries, onDeleteEntry, deletingId)}</CollapsibleContent>
    </Collapsible>
  );
}

function SourceBadge({ source }: { source: CanvasMemoryEntrySource }) {
  if (source === "manual") {
    return (
      <span className="rounded-full bg-blue-50 px-2 py-0.5 font-sans text-[11px] font-medium text-blue-700">
        Manual
      </span>
    );
  }
  if (source === "node") {
    return (
      <span className="rounded-full bg-slate-100 px-2 py-0.5 font-sans text-[11px] font-medium text-slate-600">
        Node-managed
      </span>
    );
  }
  return null;
}

function ZeroState({ canCreate, onCreate }: { canCreate: boolean; onCreate: () => void }) {
  return (
    <div
      role="status"
      className="flex min-h-0 w-full flex-1 flex-col items-center justify-center gap-4 px-6 py-16 text-center sm:px-10"
    >
      <p className="text-base font-medium text-gray-900">No canvas memory yet</p>
      <p className="max-w-lg text-pretty text-sm leading-relaxed text-gray-500">
        Use memory components on your canvas—for example <span className="font-medium text-gray-700">Add Memory</span>,{" "}
        <span className="font-medium text-gray-700">Read Memory</span>, or{" "}
        <span className="font-medium text-gray-700">Upsert Memory</span>. After a run writes to canvas memory, entries
        will show up here.
      </p>
      {canCreate ? (
        <Button type="button" size="sm" onClick={onCreate} data-testid="memory-create-bank-empty-button">
          <Plus className="h-4 w-4" aria-hidden="true" />
          Create memory bank
        </Button>
      ) : null}
    </div>
  );
}

function formatValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }

  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function collectColumns(items: Record<string, unknown>[]): string[] {
  const set = new Set<string>();
  items.forEach((item) => {
    Object.keys(item).forEach((key) => set.add(key));
  });
  return Array.from(set);
}

function renderNamespaceTable(
  values: CanvasMemoryEntry[],
  onDeleteEntry?: (memoryId: string) => void,
  deletingId?: string,
) {
  if (values.length === 0) {
    return <div className="px-3 py-2 text-xs text-gray-500">No items</div>;
  }

  const showActions = !!onDeleteEntry;
  const objectValues = values.map((entry) => entry.values).filter(isRecord) as Record<string, unknown>[];
  if (objectValues.length === values.length) {
    const columns = collectColumns(objectValues);
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-950/15 bg-slate-50">
              {columns.map((column) => (
                <th
                  key={column}
                  className="px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-gray-500"
                >
                  {column}
                </th>
              ))}
              {showActions ? (
                <th className="w-12 px-3 py-2 text-right text-[11px] font-semibold uppercase tracking-wide text-gray-500"></th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {values.map((entry, index) => {
              const item = objectValues[index];
              return (
                <tr key={entry.id || index} className="border-b border-slate-950/15 last:border-b-0">
                  {columns.map((column) => (
                    <td key={`${index}-${column}`} className="px-3 py-2 align-middle font-mono text-xs text-gray-700">
                      {formatValue(item[column])}
                    </td>
                  ))}
                  {showActions ? (
                    <td className="px-3 py-2 text-right align-middle">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        disabled={!entry.id || deletingId === entry.id}
                        onClick={() => {
                          if (entry.id) onDeleteEntry?.(entry.id);
                        }}
                        className="text-gray-500 hover:text-red-600"
                        title="Delete entry"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </td>
                  ) : null}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-950/15 bg-slate-50">
            <th className="px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-gray-500">
              Value
            </th>
            {showActions ? (
              <th className="w-12 px-3 py-2 text-right text-[11px] font-semibold uppercase tracking-wide text-gray-500"></th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {values.map((entry, index) => (
            <tr key={entry.id || index} className="border-b border-slate-950/15 last:border-b-0">
              <td className="px-3 py-2 align-middle font-mono text-xs text-gray-700">{formatValue(entry.values)}</td>
              {showActions ? (
                <td className="px-3 py-2 text-right align-middle">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    disabled={!entry.id || deletingId === entry.id}
                    onClick={() => {
                      if (entry.id) onDeleteEntry?.(entry.id);
                    }}
                    className="text-gray-500 hover:text-red-600"
                    title="Delete entry"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
