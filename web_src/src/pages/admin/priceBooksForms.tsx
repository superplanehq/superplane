import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";
import type { PriceBookVMRate } from "./priceBooksApi";

export function AddVMRateForm({
  onAdd,
  disabled = false,
}: {
  onAdd: (rate: PriceBookVMRate) => boolean;
  disabled?: boolean;
}) {
  const [matchKey, setMatchKey] = useState("");
  const [micros, setMicros] = useState("0");

  return (
    <form
      className="flex flex-col gap-3 sm:flex-row sm:items-end"
      onSubmit={(event) => {
        event.preventDefault();
        if (disabled) {
          return;
        }
        const parsed = Number.parseInt(micros, 10);
        const added = onAdd({
          match_key: matchKey.trim(),
          match_mode: "exact",
          micros_per_second: Number.isFinite(parsed) && parsed >= 0 ? parsed : 0,
        });
        if (added) {
          setMatchKey("");
          setMicros("0");
        }
      }}
    >
      <div className="min-w-40 flex-1">
        <Label htmlFor="price-book-add-vm-key">Machine type</Label>
        <Input
          id="price-book-add-vm-key"
          className="mt-1 font-mono text-xs"
          value={matchKey}
          disabled={disabled}
          onChange={(event) => setMatchKey(event.target.value)}
        />
      </div>
      <div className="w-40">
        <Label htmlFor="price-book-add-vm-micros">Micros per second</Label>
        <Input
          id="price-book-add-vm-micros"
          type="number"
          min="0"
          step="1"
          className="mt-1 text-right font-mono text-xs"
          value={micros}
          disabled={disabled}
          onChange={(event) => setMicros(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm" disabled={disabled}>
        Add machine rate
      </Button>
    </form>
  );
}
