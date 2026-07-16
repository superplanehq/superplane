import type { ComponentPropsWithoutRef, ReactNode } from "react";
import type { ExpressionAdapter, ExpressionDialectId } from "@/lib/expression";

export type ExpressionEditorDialect = ExpressionDialectId;

// - `wrapped`   — expressions wrapped in `{{ … }}` (default for template fields).
// - `singleWrapped` — a literal or exactly one full `{{ … }}` expression.
// - `raw`       — the whole input is one expression, no delimiters.
// - `pathOrRaw` — plain paths stay literal but the field also accepts a
//                 wrapped expression; the wrapped-template chrome is kept
//                 so the delimiter is visible once the user opts in.
export type ExpressionSyntaxProfile = "wrapped" | "singleWrapped" | "raw" | "pathOrRaw";

export interface ExpressionEditorProps extends Omit<ComponentPropsWithoutRef<"textarea">, "onChange" | "size"> {
  dialect?: ExpressionEditorDialect;
  // Override the default adapter selected by `dialect`.
  expressionAdapter?: ExpressionAdapter;
  syntaxProfile?: ExpressionSyntaxProfile;
  exampleObj: Record<string, unknown> | null;
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  // Escape hatches — omit to let `syntaxProfile` decide.
  expressionMode?: "wrapped" | "raw";
  startWord?: string;
  prefix?: string;
  suffix?: string;
  className?: string;
  inputSize?: "xs" | "sm" | "md" | "lg";
  minHeight?: number;
  fullHeight?: boolean;
  showValuePreview?: boolean;
  valuePreviewLabel?: string;
  quickTip?: string;
  excludedSuggestions?: string[];
  noExampleObjectText?: string;
  // Explicit override for suggesting top-level `exampleObj` keys as plain names.
  // Defaults to `true` for the `cel` dialect + `pathOrRaw` profile.
  includeTopLevelGlobals?: boolean;
  // Explicit override for suggesting the built-in expr-lang function list.
  // Defaults to `false` for the `cel` dialect, `true` otherwise.
  includeFunctions?: boolean;
}

// Note: `envKeySource` is derived from `dialect` and not user-configurable to
// keep the two in sync; widget CEL always maps `$` to `__runNodes__`.

export interface ExpressionEditorDialogChildProps {
  value: string;
  onChange: (next: string) => void;
}

export interface ExpressionEditorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  initialValue: string;
  onSave: (next: string) => void;
  testId?: string;
  children: (props: ExpressionEditorDialogChildProps) => ReactNode;
  headerActions?: (props: { draft: string }) => ReactNode;
}
