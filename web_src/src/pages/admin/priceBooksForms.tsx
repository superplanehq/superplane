import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";
import { usdInputToCents } from "./priceBookFormat";
import type { PriceBookModelRate, PriceBookVMRate } from "./priceBooksApi";

export function AddModelRateForm({
  provider,
  onAdd,
  disabled = false,
}: {
  provider: string;
  onAdd: (rate: PriceBookModelRate) => boolean;
  disabled?: boolean;
}) {
  const [matchKey, setMatchKey] = useState("");
  const [input, setInput] = useState("0.00");
  const [output, setOutput] = useState("0.00");

  return (
    <form
      className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end"
      onSubmit={(event) => {
        event.preventDefault();
        if (disabled) {
          return;
        }
        const added = onAdd({
          provider,
          match_key: matchKey.trim(),
          match_mode: "exact",
          input_cents_per_million: usdInputToCents(input),
          output_cents_per_million: usdInputToCents(output),
          cache_read_cents_per_million: 0,
          cache_write_cents_per_million: 0,
          reasoning_cents_per_million: 0,
        });
        if (added) {
          setMatchKey("");
          setInput("0.00");
          setOutput("0.00");
        }
      }}
    >
      <div className="min-w-40 flex-1">
        <Label htmlFor="price-book-add-model-key">Model</Label>
        <Input
          id="price-book-add-model-key"
          className="mt-1 font-mono text-xs"
          value={matchKey}
          disabled={disabled}
          onChange={(event) => setMatchKey(event.target.value)}
        />
      </div>
      <div className="w-28">
        <Label htmlFor="price-book-add-model-input">Input</Label>
        <Input
          id="price-book-add-model-input"
          type="number"
          min="0"
          step="0.01"
          className="mt-1 text-right"
          value={input}
          disabled={disabled}
          onChange={(event) => setInput(event.target.value)}
        />
      </div>
      <div className="w-28">
        <Label htmlFor="price-book-add-model-output">Output</Label>
        <Input
          id="price-book-add-model-output"
          type="number"
          min="0"
          step="0.01"
          className="mt-1 text-right"
          value={output}
          disabled={disabled}
          onChange={(event) => setOutput(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm" disabled={disabled}>
        Add model rate
      </Button>
    </form>
  );
}

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
