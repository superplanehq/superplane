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
    <div className="my-4 border border-amber-200 bg-amber-50 rounded-lg p-3">
      <div className="flex items-start gap-2 mb-3">
        <AlertTriangle className="size-4 text-amber-600 shrink-0 mt-0.5" />
        <p className="text-sm text-amber-900">{message}</p>
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
