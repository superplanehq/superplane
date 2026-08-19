import type { CanvasesCanvasSummary } from "@/api-client";
import { CommandItem, CommandSeparator, CommandShortcut, CommandGroup } from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { ArrowLeft, ChevronRight, FileText, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import type { PaletteAction, PalettePageAction } from "./types";

export function ActionItem({ action }: { action: PaletteAction }) {
  const Icon = action.icon;
  const value = [action.label, action.description, ...(action.keywords || [])].filter(Boolean).join(" ");

  return (
    <CommandItem
      value={value}
      disabled={action.disabled}
      onSelect={action.onSelect}
      className={cn(
        "min-h-14 cursor-pointer rounded-lg border border-transparent px-3 py-2.5 data-[selected=true]:border-border data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground",
        action.disabled && "cursor-not-allowed",
      )}
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-foreground">{action.label}</span>
        {action.description ? (
          <span className="block truncate text-xs text-muted-foreground">{action.description}</span>
        ) : null}
      </span>
      {action.shortcut ? <CommandShortcut>{action.shortcut}</CommandShortcut> : null}
    </CommandItem>
  );
}

export function PageItem({ action, onSelect }: { action: PalettePageAction; onSelect: () => void }) {
  const Icon = action.icon;
  const value = [action.label, action.description, ...(action.keywords || [])].filter(Boolean).join(" ");

  return (
    <CommandItem
      value={value}
      disabled={action.disabled}
      onSelect={onSelect}
      className={cn(
        "min-h-14 cursor-pointer rounded-lg border border-transparent px-3 py-2.5 data-[selected=true]:border-border data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground",
        action.disabled && "cursor-not-allowed",
      )}
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-foreground">{action.label}</span>
        {action.description ? (
          <span className="block truncate text-xs text-muted-foreground">{action.description}</span>
        ) : null}
      </span>
      <ChevronRight className="h-4 w-4 text-muted-foreground" />
    </CommandItem>
  );
}

export function NestedPage({ children, onBack }: { children: ReactNode; onBack: () => void }) {
  return (
    <>
      <CommandGroup>
        <CommandItem
          value="back return previous"
          onSelect={onBack}
          className="min-h-11 cursor-pointer rounded-lg px-3 py-2 data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
        >
          <ArrowLeft className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">Back to commands</span>
        </CommandItem>
      </CommandGroup>
      <CommandSeparator className="my-2" />
      {children}
    </>
  );
}

export function CanvasListItems({
  canvases,
  canvasesLoading,
  description,
  emptyLabel,
  icon,
  onSelect,
}: {
  canvases: CanvasesCanvasSummary[];
  canvasesLoading: boolean;
  description: string;
  emptyLabel: string;
  icon: LucideIcon;
  onSelect: (canvas: CanvasesCanvasSummary) => void;
}) {
  if (canvasesLoading) {
    return (
      <CommandItem disabled value="loading canvases" className="min-h-12 rounded-lg px-3 py-2.5">
        <FileText className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">Loading canvases...</span>
      </CommandItem>
    );
  }

  if (canvases.length === 0) {
    return (
      <CommandItem disabled value="no canvases" className="min-h-12 rounded-lg px-3 py-2.5">
        <FileText className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">{emptyLabel}</span>
      </CommandItem>
    );
  }

  return canvases
    .filter((canvas) => canvas.id)
    .map((canvas) => (
      <ActionItem
        key={canvas.id}
        action={{
          id: canvas.id || "",
          label: canvas.name || "Untitled canvas",
          description,
          icon,
          onSelect: () => onSelect(canvas),
          keywords: [canvas.description || ""],
        }}
      />
    ));
}
