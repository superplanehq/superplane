import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { usdInputToCents } from "./priceBookFormat";
import type { PriceBookModelRate, PriceBookVMRate } from "./priceBooksApi";

export function AddModelRateForm({ onAdd }: { onAdd: (rate: PriceBookModelRate) => boolean }) {
  const [matchKey, setMatchKey] = useState("");
  const [matchMode, setMatchMode] = useState("prefix");
  const [input, setInput] = useState("0.00");
  const [output, setOutput] = useState("0.00");

  return (
    <form
      className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-end"
      onSubmit={(event) => {
        event.preventDefault();
        const added = onAdd({
          match_key: matchKey.trim(),
          match_mode: matchMode,
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
        <Label htmlFor="price-book-add-model-key">Match key</Label>
        <Input
          id="price-book-add-model-key"
          className="mt-1 font-mono text-xs"
          value={matchKey}
          onChange={(event) => setMatchKey(event.target.value)}
        />
      </div>
      <div className="w-40">
        <Label htmlFor="price-book-add-model-mode">Mode</Label>
        <Select value={matchMode} onValueChange={setMatchMode}>
          <SelectTrigger id="price-book-add-model-mode" className="mt-1">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="prefix">Prefix</SelectItem>
            <SelectItem value="family">Family</SelectItem>
          </SelectContent>
        </Select>
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
          onChange={(event) => setOutput(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm">
        Add model rate
      </Button>
    </form>
  );
}

export function AddVMRateForm({ onAdd }: { onAdd: (rate: PriceBookVMRate) => boolean }) {
  const [matchKey, setMatchKey] = useState("");
  const [micros, setMicros] = useState("0");

  return (
    <form
      className="flex flex-col gap-3 sm:flex-row sm:items-end"
      onSubmit={(event) => {
        event.preventDefault();
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
          onChange={(event) => setMicros(event.target.value)}
        />
      </div>
      <Button type="submit" variant="outline" size="sm">
        Add VM rate
      </Button>
    </form>
  );
}
