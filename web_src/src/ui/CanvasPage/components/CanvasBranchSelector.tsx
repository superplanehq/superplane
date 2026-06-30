import type { CanvasesCanvasBranch } from "@/api-client";
import { Command, CommandEmpty, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { branchName, sortCanvasBranches } from "@/lib/canvas-branches";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Command as CommandPrimitive } from "cmdk";
import { Check, GitBranch, Search } from "lucide-react";
import { useMemo, useState } from "react";

const SWITCHER_HEIGHT_CLASS = "h-7";
const SWITCHER_BORDER_CLASS = "border border-slate-950/20";
const SWITCHER_MENU_SURFACE_CLASS = "rounded-md bg-white shadow-md";
const SEARCH_ROW_CLASS_NAME = cn(
  "flex items-center gap-2 bg-white px-2.5 text-[13px]",
  SWITCHER_HEIGHT_CLASS,
  "rounded-none border-0 border-b border-slate-950/10 ring-0",
);
const TRIGGER_SURFACE_CLASS_NAME = cn(
  "rounded-md bg-transparent transition-colors hover:bg-slate-100",
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/40",
  "disabled:pointer-events-none disabled:opacity-50",
);
const TRIGGER_ICON_CLASS_NAME = "size-3.5 shrink-0 text-slate-400 transition-colors group-hover/trigger:text-slate-800";
const TRIGGER_CLASS_NAME = cn(
  "group/trigger flex w-full items-center gap-2 px-2.5 text-[13px]",
  SWITCHER_HEIGHT_CLASS,
  "justify-center",
  TRIGGER_SURFACE_CLASS_NAME,
);

type CanvasBranchSelectorProps = {
  branches: CanvasesCanvasBranch[];
  value: string;
  onValueChange: (branchName: string) => void;
  disabled?: boolean;
};

export function CanvasBranchSelector({ branches, value, onValueChange, disabled }: CanvasBranchSelectorProps) {
  const [open, setOpen] = useState(false);
  const sortedBranches = useMemo(() => sortCanvasBranches(branches), [branches]);
  const displayName = value.trim() || "Branch";
  const isDisabled = disabled || sortedBranches.length === 0;

  const handleSelect = (nextBranchName: string) => {
    setOpen(false);
    if (nextBranchName === value) {
      return;
    }
    onValueChange(nextBranchName);
  };

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (isDisabled) {
          return;
        }
        setOpen(nextOpen);
      }}
    >
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Switch branch"
          aria-expanded={open}
          disabled={isDisabled}
          data-testid="canvas-branch-selector"
          className={cn(TRIGGER_CLASS_NAME, open && "pointer-events-none invisible")}
        >
          <GitBranch className={TRIGGER_ICON_CLASS_NAME} aria-hidden="true" />
          <span className="min-w-0 truncate font-medium text-slate-800">{displayName}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        side="bottom"
        sideOffset={-28}
        className={cn(
          SWITCHER_MENU_SURFACE_CLASS,
          SWITCHER_BORDER_CLASS,
          "w-[var(--radix-popover-trigger-width)] overflow-hidden p-0 outline-none",
        )}
      >
        <Command shouldFilter className="bg-white">
          <div className={SEARCH_ROW_CLASS_NAME}>
            <Search className="size-3.5 shrink-0 text-slate-400" aria-hidden="true" />
            <CommandPrimitive.Input
              placeholder="Search branches"
              className="min-w-0 flex-1 bg-transparent text-[13px] font-medium text-slate-700 outline-none placeholder:font-normal placeholder:text-slate-400"
            />
          </div>
          <CommandList className="max-h-[280px]">
            <CommandEmpty className="py-6 text-[13px]">No branches found.</CommandEmpty>
            <CommandGroup heading="Branches" className="text-[13px]">
              {sortedBranches.map((branch) => {
                const name = branchName(branch);
                if (!name) {
                  return null;
                }

                return (
                  <CommandItem
                    key={branch.id ?? name}
                    value={name}
                    keywords={[name]}
                    onSelect={() => handleSelect(name)}
                    className="cursor-pointer text-[13px] data-[selected=true]:bg-sky-100 data-[selected=true]:text-slate-900"
                  >
                    <span className="min-w-0 flex-1 truncate">{name}</span>
                    {name === value ? <Check className="size-3.5 shrink-0 text-slate-600" /> : null}
                  </CommandItem>
                );
              })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
