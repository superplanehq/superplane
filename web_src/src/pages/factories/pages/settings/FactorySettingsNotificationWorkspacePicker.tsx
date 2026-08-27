import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Plus, X } from "lucide-react";
import { useMemo, useState } from "react";

interface FactoryOption {
  id?: string;
  name?: string;
}

export function FactorySettingsNotificationWorkspacePicker({
  factories,
  selectedFactoryIds,
  onAdd,
  onRemove,
}: {
  factories: FactoryOption[];
  selectedFactoryIds: string[];
  onAdd: (factoryId: string) => void;
  onRemove: (factoryId: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const factoriesById = useMemo(() => {
    const map = new Map<string, FactoryOption>();
    for (const factory of factories) {
      if (factory.id) {
        map.set(factory.id, factory);
      }
    }
    return map;
  }, [factories]);

  const available = factories.filter((factory) => factory.id && !selectedFactoryIds.includes(factory.id));

  if (factories.length === 0) {
    return <p className="text-[12px] text-muted-foreground">No workspaces available.</p>;
  }

  return (
    <div
      className="flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1.5"
      data-testid="notifications-workspace-picker"
    >
      {selectedFactoryIds.map((factoryId) => {
        const name = factoriesById.get(factoryId)?.name || factoryId;
        return (
          <Badge key={factoryId} variant="secondary" className="gap-1 pr-1 font-normal">
            <span className="max-w-40 truncate">{name}</span>
            <button
              type="button"
              className="rounded-sm p-0.5 text-muted-foreground hover:bg-background/60 hover:text-foreground"
              aria-label={`Remove ${name}`}
              data-testid={`notifications-factory-${factoryId}`}
              onClick={() => onRemove(factoryId)}
            >
              <X className="size-3" aria-hidden />
            </button>
          </Badge>
        );
      })}
      {available.length > 0 ? (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="h-6 gap-1 px-2 text-[12px] text-muted-foreground"
              data-testid="notifications-add-workspace"
            >
              <Plus className="size-3" aria-hidden />
              Select workspace
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-72 p-0" sideOffset={6}>
            <Command>
              <CommandInput placeholder="Find a workspace" />
              <CommandList>
                <CommandEmpty>No workspace found.</CommandEmpty>
                <CommandGroup>
                  {available.map((factory) => {
                    const factoryId = factory.id ?? "";
                    return (
                      <CommandItem
                        key={factoryId}
                        value={`${factory.name ?? ""} ${factoryId}`}
                        data-testid={`notifications-factory-option-${factoryId}`}
                        onSelect={() => onAdd(factoryId)}
                      >
                        {factory.name || factoryId}
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      ) : null}
    </div>
  );
}
