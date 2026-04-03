import { Button } from "@/components/ui/button";

export interface SuggestedValuePanelProps {
  proposedValue: string;
  explanation: string | null;
  onDiscard: () => void;
  onUseValue: () => void;
}

export function SuggestedValuePanel({ proposedValue, explanation, onDiscard, onUseValue }: SuggestedValuePanelProps) {
  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-muted-foreground">Suggested value</p>
      <pre className="max-h-40 overflow-auto rounded-md bg-muted/50 p-2 text-xs whitespace-pre-wrap break-all">
        {proposedValue}
      </pre>
      {explanation ? <p className="text-xs text-muted-foreground">{explanation}</p> : null}
      <div className="flex justify-end gap-2 pt-1">
        <Button type="button" size="sm" variant="outline" onClick={onDiscard}>
          Discard
        </Button>
        <Button type="button" size="sm" onClick={onUseValue}>
          Use this value
        </Button>
      </div>
    </div>
  );
}
