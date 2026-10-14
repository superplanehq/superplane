import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePermissions } from "@/contexts/usePermissions";
import { Sheet, SheetContent, SheetDescription, SheetTitle, SheetTrigger } from "@/ui/sheet";
import { Check, ChevronDown, Plus, Search } from "lucide-react";
import { useId, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { initialsForName } from "../layout/factoriesRail";
import { factoryRouteSegment, newFactoryPath, pathAfterWorkspaceSwitch } from "../lib/factoryPagePaths";

export function MobileWorkspaceSwitcher() {
  const { organizationId, routeSegment, factory, factories } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const searchId = useId();
  const name = factory?.name?.trim() || "Workspace";
  const choices = factories.filter((entry) =>
    (entry.name ?? "").toLocaleLowerCase().includes(search.trim().toLocaleLowerCase()),
  );

  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setSearch("");
      }}
    >
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          className="h-12 max-w-full min-w-0 justify-start gap-3 px-0 hover:bg-transparent"
          aria-label={`Switch workspace, ${name}`}
        >
          <span
            className="flex size-9 shrink-0 items-center justify-center rounded-md bg-sidebar-accent text-sm font-medium text-foreground"
            aria-hidden
          >
            {initialsForName(name)}
          </span>
          <span className="truncate text-base font-semibold">{name}</span>
          <ChevronDown className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        </Button>
      </SheetTrigger>
      <SheetContent
        side="bottom"
        className="max-h-[85dvh] overflow-y-auto rounded-t-lg px-4 pt-6 pb-[max(1.5rem,env(safe-area-inset-bottom))] [&>button.absolute]:size-11 [&>button.absolute]:top-3 [&>button.absolute]:right-3 [&>button.absolute]:flex [&>button.absolute]:items-center [&>button.absolute]:justify-center"
      >
        <SheetTitle className="pr-12 text-left text-xl">Switch workspace</SheetTitle>
        <SheetDescription className="sr-only">Select a workspace to open.</SheetDescription>
        <div className="relative mt-5">
          <Label htmlFor={searchId} className="sr-only">
            Search workspaces
          </Label>
          <Search className="absolute left-3 top-3.5 size-4 text-muted-foreground" aria-hidden />
          <Input
            id={searchId}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search workspaces"
            className="h-11 pl-10 text-base"
          />
        </div>
        <div className="mt-3 max-h-[45dvh] space-y-2 overflow-y-auto">
          {choices.map((entry) => {
            const current = entry.id === factory?.id;
            const entryName = entry.name?.trim() || "Workspace";
            return (
              <Button
                key={entry.id}
                variant="ghost"
                disabled={!factoryRouteSegment(entry)}
                aria-current={current ? "true" : undefined}
                className="h-16 w-full justify-start gap-3 rounded-md hover:bg-accent px-3 text-left"
                onClick={() => {
                  if (!current)
                    navigate(
                      pathAfterWorkspaceSwitch({
                        pathname,
                        organizationId,
                        currentFactoryKey: routeSegment,
                        nextFactory: entry,
                      }),
                    );
                  setOpen(false);
                  setSearch("");
                }}
              >
                <span
                  className="flex size-10 shrink-0 items-center justify-center rounded-md bg-sidebar-accent text-foreground"
                  aria-hidden
                >
                  {initialsForName(entryName)}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-base">{entryName}</span>
                  {current ? (
                    <span className="block text-xs font-normal text-muted-foreground">Current workspace</span>
                  ) : null}
                </span>
                {current ? <Check className="size-5 text-foreground" aria-hidden /> : null}
              </Button>
            );
          })}
          {choices.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No matching workspaces.</p>
          ) : null}
        </div>
        {canAct("factories", "create") ? (
          <Button
            variant="ghost"
            className="mt-4 h-12 w-full justify-start gap-3 border-t border-border"
            onClick={() => {
              setOpen(false);
              navigate(newFactoryPath(organizationId));
            }}
          >
            <Plus className="size-5" aria-hidden />
            Create workspace
          </Button>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
