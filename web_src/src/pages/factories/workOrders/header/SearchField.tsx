import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Search, X } from "lucide-react";
import type { KeyboardEvent, RefObject } from "react";

interface SearchFieldProps {
  inputRef: RefObject<HTMLInputElement | null>;
  open: boolean;
  value: string;
  onOpen: () => void;
  onChange: (value: string) => void;
  onClose: () => void;
}

/** Collapses to an icon and expands in place, so the title bar stays quiet. */
export function SearchField({ inputRef, open, value, onOpen, onChange, onClose }: SearchFieldProps) {
  if (!open) {
    return (
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={onOpen}
        aria-label="Search tasks"
        className="size-8 shrink-0 text-muted-foreground"
        data-testid="work-orders-search-trigger"
      >
        <Search className="size-3.5" aria-hidden />
      </Button>
    );
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
    }
  };

  return (
    <div className="relative">
      <Search
        className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
        aria-hidden
      />
      <Input
        ref={inputRef}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Search…"
        aria-label="Search tasks"
        className="!h-8 w-[200px] pl-8 pr-8 text-[13px] shadow-none"
        data-testid="work-orders-search-input"
      />
      <button
        type="button"
        onClick={onClose}
        aria-label="Close search"
        className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
        data-testid="work-orders-search-close"
      >
        <X className="size-3.5" aria-hidden />
      </button>
    </div>
  );
}
