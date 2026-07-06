import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ConfirmWidgetProps {
  message: string;
  yes: string;
  no: string;
  onAction?: (text: string) => void;
}

export function ConfirmWidget({ message, yes, no, onAction }: ConfirmWidgetProps) {
  return (
    <div className="my-4 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/60 dark:bg-amber-950/40">
      <div className="mb-3 flex items-start gap-2">
        <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
        <p className="text-sm text-amber-900 dark:text-amber-100">{message}</p>
      </div>
      <div className="flex gap-2">
        <Button size="sm" variant="destructive" className="text-xs" onClick={() => onAction?.(yes)}>
          {yes}
        </Button>
        <Button size="sm" variant="outline" className="text-xs" onClick={() => onAction?.(no)}>
          {no}
        </Button>
      </div>
    </div>
  );
}
