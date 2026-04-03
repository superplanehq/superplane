import { useCallback, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { RefObject } from "react";
import { Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { SuggestFieldValueFn } from "@/ui/configurationFieldRenderer/types";
import { twMerge } from "tailwind-merge";

export interface InlineFieldAssistantProps {
  fieldLabel: string;
  onApplyValue: (value: string) => void;
  suggestFieldValue?: SuggestFieldValueFn;
  labelRightRef?: RefObject<HTMLDivElement | null>;
  labelRightReady?: boolean;
}

export function InlineFieldAssistant({
  fieldLabel,
  onApplyValue,
  suggestFieldValue,
  labelRightRef,
  labelRightReady = false,
}: InlineFieldAssistantProps) {
  const [open, setOpen] = useState(false);
  const [instruction, setInstruction] = useState("");
  const [proposedValue, setProposedValue] = useState<string | null>(null);
  const [explanation, setExplanation] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const generationRef = useRef(0);

  const resetPanel = useCallback(() => {
    setInstruction("");
    setProposedValue(null);
    setExplanation(null);
    setError(null);
    setLoading(false);
  }, []);

  const handleClose = useCallback(() => {
    generationRef.current += 1;
    setOpen(false);
    resetPanel();
  }, [resetPanel]);

  const handleGenerate = useCallback(async () => {
    if (!suggestFieldValue) return;
    const gen = ++generationRef.current;
    setLoading(true);
    setError(null);
    setProposedValue(null);
    setExplanation(null);
    try {
      const out = await suggestFieldValue(instruction);
      if (generationRef.current !== gen) return;
      setProposedValue(out.value);
      setExplanation(out.explanation ?? null);
    } catch (e) {
      if (generationRef.current !== gen) return;
      setError(e instanceof Error ? e.message : "Something went wrong");
    } finally {
      if (generationRef.current === gen) {
        setLoading(false);
      }
    }
  }, [instruction, suggestFieldValue]);

  const handleConfirm = useCallback(() => {
    if (proposedValue == null) return;
    onApplyValue(proposedValue);
    handleClose();
  }, [proposedValue, onApplyValue, handleClose]);

  if (!suggestFieldValue) {
    return null;
  }

  const trigger = (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground"
      aria-label={`Open assistant for ${fieldLabel}`}
      aria-expanded={open}
      disabled={open && loading}
      onClick={() => (open ? handleClose() : setOpen(true))}
    >
      <Sparkles className="h-4 w-4" />
    </Button>
  );

  const panel = open ? (
    <div
      className={twMerge("rounded-md border border-border bg-card p-3 shadow-sm space-y-3", "ring-1 ring-ring/10")}
      role="region"
      aria-label={`Assistant for ${fieldLabel}`}
    >
      <Textarea
        value={instruction}
        onChange={(e) => setInstruction(e.target.value)}
        placeholder="Describe what you want this field to contain…"
        className="min-h-[72px] resize-y"
        disabled={loading}
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" size="sm" onClick={() => void handleGenerate()} disabled={loading}>
          {loading ? "Generating…" : "Generate"}
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={handleClose} disabled={loading}>
          Cancel
        </Button>
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {proposedValue != null ? (
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">Suggested value</p>
          <pre className="max-h-40 overflow-auto rounded-md bg-muted/50 p-2 text-xs whitespace-pre-wrap break-all">
            {proposedValue}
          </pre>
          {explanation ? <p className="text-xs text-muted-foreground">{explanation}</p> : null}
          <div className="flex justify-end gap-2 pt-1">
            <Button type="button" size="sm" variant="outline" onClick={handleClose}>
              Discard
            </Button>
            <Button type="button" size="sm" onClick={handleConfirm}>
              Use this value
            </Button>
          </div>
        </div>
      ) : !loading && !error ? (
        <p className="text-xs text-muted-foreground">The result will show up here.</p>
      ) : null}
    </div>
  ) : null;

  return (
    <>
      {labelRightReady && labelRightRef?.current ? (
        createPortal(trigger, labelRightRef.current)
      ) : (
        <div className="flex justify-end mb-1">{trigger}</div>
      )}
      {panel}
    </>
  );
}
