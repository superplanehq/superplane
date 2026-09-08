import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Check, Triangle } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";

import { factoriesRailControlClassName, initialsForName } from "../../../layout/factoriesRail";
import { factoryHomePath, firstFactoryLineId } from "../../../lib/factoryPagePaths";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunChrome, FirstRunWorkspaceOption } from "./firstRunTypes";

function workspaceLabel(factory: FirstRunWorkspaceOption): string {
  return factory.name?.trim() || "Workspace";
}

export function FirstRunWorkspaceSwitch({
  switcher,
}: {
  switcher: NonNullable<FirstRunChrome["workspaceSwitch"]> | undefined;
}) {
  if (!switcher) return null;
  return <FirstRunWorkspaceSwitchMenu switcher={switcher} />;
}

function FirstRunWorkspaceSwitchMenu({
  switcher,
}: {
  switcher: NonNullable<FirstRunChrome["workspaceSwitch"]>;
}) {
  const navigate = useNavigate();
  const copy = FIRST_RUN_COPY.chrome;
  const [open, setOpen] = useState(false);
  const current = switcher.factories.find((factory) => factory.id === switcher.currentFactoryId);
  const currentName = current ? workspaceLabel(current) : "Workspace";

  return (
    <div className="pointer-events-auto absolute bottom-0 left-0 z-10 px-6 pb-6">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label={`${copy.switchWorkspace}, ${currentName}`}
            title={currentName}
            className={cn(
              factoriesRailControlClassName,
              "bg-muted text-[11px] font-medium tracking-[-0.01em] text-foreground hover:bg-accent",
            )}
            data-testid="first-run-workspace-switch"
          >
            {initialsForName(currentName)}
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" side="top" sideOffset={8} className="w-64 p-1">
          <p className="px-2 py-1.5 text-sm font-medium">{copy.switchWorkspace}</p>
          {switcher.factories.map((factory) => {
            if (!factory.id) return null;
            const isCurrent = factory.id === switcher.currentFactoryId;
            return (
              <button
                key={factory.id}
                type="button"
                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent hover:text-accent-foreground"
                onClick={() => {
                  setOpen(false);
                  if (isCurrent || !factory.key) return;
                  navigate(factoryHomePath(switcher.organizationId, factory.key, firstFactoryLineId(factory)));
                }}
                data-testid={`first-run-workspace-option-${factory.id}`}
              >
                <Triangle className="h-3.5 w-3.5 shrink-0" aria-hidden />
                <span className="truncate">{workspaceLabel(factory)}</span>
                {isCurrent ? <Check className="ml-auto h-3.5 w-3.5 shrink-0" aria-hidden /> : null}
              </button>
            );
          })}
        </PopoverContent>
      </Popover>
    </div>
  );
}
