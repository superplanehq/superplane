import type { CanvasesResolvedReplayInputStatus } from "@/api-client";
import { ReplayInputEditor } from "./ReplayInputEditor";

export type ReplayInputListItem = {
  testIdPrefix: string;
  label: string;
  editedText: string;
  originalText: string;
  status?: CanvasesResolvedReplayInputStatus;
  error?: string | null;
  errorKind?: "invalid-json" | "missing" | null;
};

export function ReplayInputsList({
  inputs,
  monacoTheme,
  readOnly,
  onChange,
}: {
  inputs: ReplayInputListItem[];
  monacoTheme: string;
  readOnly: boolean;
  onChange: (index: number, value: string) => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto" data-testid="replay-inputs-list">
      {inputs.map((input, index) => (
        // Keyed by position, not by source: one source can contribute several inputs,
        // and they must not share an edit slot.
        <ReplayInputEditor
          key={index}
          testIdPrefix={input.testIdPrefix}
          label={input.label}
          editedText={input.editedText}
          originalText={input.originalText}
          monacoTheme={monacoTheme}
          status={input.status}
          readOnly={readOnly}
          error={input.error}
          errorKind={input.errorKind}
          onChange={(value) => onChange(index, value)}
        />
      ))}
    </div>
  );
}
