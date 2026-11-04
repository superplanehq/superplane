import { useState } from "react";
import { Check, ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

export function BranchPicker(props: {
  id: string;
  branch: string;
  choices: string[];
  disabled: boolean;
  onChange: (branch: string) => void;
  /** Label of an extra first choice that clears the branch. */
  emptyChoiceLabel?: string;
}) {
  const [open, setOpen] = useState(false);
  const selectBranch = (name: string) => {
    props.onChange(name);
    setOpen(false);
  };
  const label = props.branch || props.emptyChoiceLabel || "Select a branch";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          id={props.id}
          role="combobox"
          aria-expanded={open}
          aria-haspopup="listbox"
          disabled={props.disabled}
          data-testid={props.id}
          className="w-full justify-between rounded-md px-2.5 font-normal"
        >
          <span
            className={cn(
              "min-w-0 truncate",
              props.branch || props.emptyChoiceLabel ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {label}
          </span>
          <ChevronDown className="size-4 shrink-0 opacity-50" aria-hidden />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--radix-popover-trigger-width)] p-0">
        <Command>
          <CommandInput placeholder="Find a branch" autoFocus />
          <CommandList>
            <CommandEmpty>No branch found.</CommandEmpty>
            <CommandGroup>
              {props.emptyChoiceLabel ? (
                <BranchChoice
                  name={props.emptyChoiceLabel}
                  selected={!props.branch}
                  onSelect={() => selectBranch("")}
                />
              ) : null}
              {props.choices.map((name) => (
                <BranchChoice
                  key={name}
                  name={name}
                  selected={name === props.branch}
                  onSelect={() => selectBranch(name)}
                />
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function BranchChoice({ name, selected, onSelect }: { name: string; selected: boolean; onSelect: () => void }) {
  return (
    <CommandItem value={name} onSelect={onSelect}>
      <span className="min-w-0 flex-1 truncate">{name}</span>
      <Check className={cn("size-4", selected ? "opacity-100" : "opacity-0")} aria-hidden />
    </CommandItem>
  );
}
