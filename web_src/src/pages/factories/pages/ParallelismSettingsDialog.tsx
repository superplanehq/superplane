import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { useEffect, useState } from "react";

import {
  clampLineStepParallelism,
  LINE_STEP_PARALLELISM_MAX,
  LINE_STEP_PARALLELISM_MIN,
} from "../lib/factoryLineFormShared";

interface ParallelismSettingsDialogProps {
  open: boolean;
  value: number;
  onSave: (value: number) => void;
  onClose: () => void;
}

const VALUE_ERROR = "Enter a whole number from 1 to 100.";

function parseParallelism(raw: string): { value: number; error?: string } {
  const trimmed = raw.trim();
  if (!/^\d+$/.test(trimmed)) {
    return { value: LINE_STEP_PARALLELISM_MIN, error: VALUE_ERROR };
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (parsed < LINE_STEP_PARALLELISM_MIN || parsed > LINE_STEP_PARALLELISM_MAX) {
    return { value: parsed, error: VALUE_ERROR };
  }
  return { value: parsed };
}

export function ParallelismSettingsDialog({ open, value, onSave, onClose }: ParallelismSettingsDialogProps) {
  const [draft, setDraft] = useState(value);
  const [input, setInput] = useState(String(value));
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    setDraft(value);
    setInput(String(value));
    setError("");
  }, [open, value]);

  const applyValue = (next: number) => {
    const clamped = clampLineStepParallelism(next);
    setDraft(clamped);
    setInput(String(clamped));
    setError("");
  };

  const handleSave = () => {
    const parsed = parseParallelism(input);
    if (parsed.error) {
      setError(parsed.error);
      return;
    }
    onSave(parsed.value);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md" data-testid="lines-parallelism-settings">
        <DialogHeader>
          <DialogTitle>Set parallelism</DialogTitle>
          <DialogDescription>Maximum number of work orders to be processed in parallel at this step.</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="lines-parallelism-input">Parallelism</Label>
            <Input
              id="lines-parallelism-input"
              data-testid="lines-parallelism-input"
              type="number"
              min={LINE_STEP_PARALLELISM_MIN}
              max={LINE_STEP_PARALLELISM_MAX}
              inputMode="numeric"
              value={input}
              onChange={(event) => {
                const raw = event.target.value;
                setInput(raw);
                setError("");
                const parsed = parseParallelism(raw);
                if (!parsed.error) {
                  setDraft(parsed.value);
                }
              }}
            />
          </div>
          <Slider
            min={LINE_STEP_PARALLELISM_MIN}
            max={LINE_STEP_PARALLELISM_MAX}
            step={1}
            value={[draft]}
            onValueChange={(values) => applyValue(values[0] ?? draft)}
            aria-label="Parallelism slider"
            data-testid="lines-parallelism-slider"
          />
          {error ? <p className="text-xs text-red-600">{error}</p> : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} data-testid="lines-parallelism-save">
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
