import { resolveIcon } from "@/lib/utils";
import type { useContextMenu } from "@/hooks/useContextMenu";
import type { BuildingBlock, BuildingBlockCategory } from "@/ui/BuildingBlocksSidebar";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { ChevronLeft, Copy, Database, Layers3, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo } from "react";

export type CanvasContextMenuData =
  | {
      kind: "canvas";
      flowPosition: { x: number; y: number };
    }
  | {
      kind: "node";
      nodeId: string;
    };

type ContextMenuGroup = {
  key: string;
  title: string;
  iconSrc?: string;
  iconSlug?: "layers-3" | "database" | "plus";
  blocks: BuildingBlock[];
};

interface CanvasContextMenuProps {
  isOpen: boolean;
  position: { x: number; y: number } | null;
  contextMenuData: CanvasContextMenuData | null;
  buildingBlocks: BuildingBlockCategory[];
  selectedGroupKey: string | null;
  onSelectGroupKey: (groupKey: string | null) => void;
  onSelectBlock: (block: BuildingBlock) => void | Promise<void>;
  onCopy: () => void;
  onEdit: () => void;
  onDelete: () => void;
  canCopy: boolean;
  canDelete: boolean;
  menuRef: ReturnType<typeof useContextMenu<CanvasContextMenuData>>["menuRef"];
  backdropProps: ReturnType<typeof useContextMenu<CanvasContextMenuData>>["backdropProps"];
  menuProps: ReturnType<typeof useContextMenu<CanvasContextMenuData>>["menuProps"];
}

function getBuildingBlockLabel(block: BuildingBlock): string {
  return block.label || block.name;
}

function getContextMenuGroup(category: BuildingBlockCategory): ContextMenuGroup {
  const normalizedName = category.name.trim().toLowerCase();

  if (normalizedName === "core") {
    return {
      key: category.name,
      title: category.name,
      iconSlug: "layers-3",
      blocks: category.blocks,
    };
  }

  if (normalizedName === "memory") {
    return {
      key: category.name,
      title: category.name,
      iconSlug: "database",
      blocks: category.blocks,
    };
  }

  const integrationName = category.blocks[0]?.integrationName;

  return {
    key: category.name,
    title: category.name,
    iconSrc: getIntegrationIconSrc(integrationName),
    iconSlug: "plus",
    blocks: category.blocks,
  };
}

export function CanvasContextMenu({
  isOpen,
  position,
  contextMenuData,
  buildingBlocks,
  selectedGroupKey,
  onSelectGroupKey,
  onSelectBlock,
  onCopy,
  onEdit,
  onDelete,
  canCopy,
  canDelete,
  menuRef,
  backdropProps,
  menuProps,
}: CanvasContextMenuProps) {
  const contextMenuGroups = useMemo(() => buildingBlocks.map(getContextMenuGroup), [buildingBlocks]);
  const selectedContextMenuGroup = useMemo(
    () => contextMenuGroups.find((group) => group.key === selectedGroupKey) || null,
    [contextMenuGroups, selectedGroupKey],
  );

  if (!isOpen || !position) {
    return null;
  }

  return (
    <>
      <div
        className="fixed inset-0 z-[55]"
        {...backdropProps}
      />
      <div
        ref={menuRef}
        className="fixed z-[60] min-w-56 rounded-xl border border-slate-200 bg-white/95 p-1.5 shadow-xl backdrop-blur-sm"
        style={{
          left: position.x,
          top: position.y,
        }}
        {...menuProps}
      >
        {contextMenuData?.kind === "canvas" ? (
          <div className="max-h-[28rem] w-80 overflow-y-auto">
            {selectedContextMenuGroup ? (
              <>
                <div className="flex items-center gap-1 px-1 pb-1">
                  <button
                    type="button"
                    onClick={() => onSelectGroupKey(null)}
                    className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm font-medium text-slate-600 transition hover:bg-slate-100 hover:text-slate-900"
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Back
                  </button>
                  <div className="truncate px-1 text-sm font-semibold text-slate-900">
                    {selectedContextMenuGroup.title}
                  </div>
                </div>
                <div className="space-y-0.5">
                  {selectedContextMenuGroup.blocks.map((block) => {
                    const BlockIcon = resolveIcon(block.icon);
                    return (
                      <button
                        key={`${selectedContextMenuGroup.key}-${block.id || block.name}`}
                        type="button"
                        onClick={() => void onSelectBlock(block)}
                        className="flex w-full items-start gap-2 rounded-lg px-2 py-2 text-left transition hover:bg-slate-100"
                      >
                        <span className="mt-0.5 text-slate-500">
                          {block.icon ? <BlockIcon size={16} /> : <Plus className="h-4 w-4" />}
                        </span>
                        <span className="min-w-0">
                          <span className="block truncate text-sm font-medium text-slate-900">
                            {getBuildingBlockLabel(block)}
                          </span>
                          {block.description ? (
                            <span className="block text-xs text-slate-500">{block.description}</span>
                          ) : null}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </>
            ) : (
              <>
                <div className="px-2 py-1.5 text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">
                  Add building block
                </div>
                <div className="space-y-1">
                  {contextMenuGroups.map((group) => {
                    const GroupIcon = group.iconSlug ? resolveIcon(group.iconSlug) : null;
                    return (
                      <button
                        key={group.key}
                        type="button"
                        onClick={() => onSelectGroupKey(group.key)}
                        className="flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left transition hover:bg-slate-100"
                      >
                        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100">
                          {group.iconSrc ? (
                            <img src={group.iconSrc} alt={group.title} className="h-5 w-5 object-contain" />
                          ) : GroupIcon ? (
                            <GroupIcon size={18} className="text-slate-600" />
                          ) : group.title === "Memory" ? (
                            <Database className="h-4 w-4 text-slate-600" />
                          ) : (
                            <Layers3 className="h-4 w-4 text-slate-600" />
                          )}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-medium text-slate-900">{group.title}</span>
                          <span className="block text-xs text-slate-500">{group.blocks.length} available</span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              </>
            )}
          </div>
        ) : null}
        {contextMenuData?.kind === "node" ? (
          <div className="w-full min-w-56">
            <button
              type="button"
              onClick={onCopy}
              disabled={!canCopy}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm text-slate-700 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Copy className="h-4 w-4" />
              Copy
            </button>
            <button
              type="button"
              onClick={onEdit}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm text-slate-700 transition hover:bg-slate-100"
            >
              <Pencil className="h-4 w-4" />
              Edit
            </button>
            <button
              type="button"
              onClick={onDelete}
              disabled={!canDelete}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm text-red-600 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </button>
          </div>
        ) : null}
      </div>
    </>
  );
}
