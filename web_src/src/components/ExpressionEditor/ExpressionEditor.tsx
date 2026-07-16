import { forwardRef, useMemo } from "react";
import { exprLangAdapter } from "@/lib/expression";
import { AutoCompleteInput, type AutoCompleteInputProps } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { getExpressionDialectAdapter } from "./expressionDialectRegistry";
import type { ExpressionEditorDialect, ExpressionEditorProps, ExpressionSyntaxProfile } from "./types";

interface ProfileDefaults {
  expressionMode: AutoCompleteInputProps["expressionMode"];
  startWord?: string;
  prefix?: string;
  suffix?: string;
}

const PROFILE: Record<ExpressionSyntaxProfile, ProfileDefaults> = {
  wrapped: { expressionMode: "wrapped", startWord: "{{", prefix: "{{ ", suffix: " }}" },
  raw: { expressionMode: "raw" },
  pathOrRaw: { expressionMode: "wrapped", startWord: "{{", prefix: "{{ ", suffix: " }}" },
};

const QUICK_TIP: Record<ExpressionEditorDialect, Record<ExpressionSyntaxProfile, string>> = {
  "expr-lang": {
    wrapped: "Tip: type `{{` to start an expression.",
    raw: "Tip: type `$` to browse node payloads.",
    pathOrRaw: "Tip: type `{{` to switch to an expression.",
  },
  cel: {
    wrapped: "Tip: type `{{` to start a CEL expression.",
    raw: "Tip: reference row fields directly (e.g. `field.name`).",
    pathOrRaw: "Tip: type `{{` to switch to a CEL expression.",
  },
};

export const ExpressionEditor = forwardRef<HTMLTextAreaElement, ExpressionEditorProps>(
  function ExpressionEditorRender(props, forwardedRef) {
    const {
      dialect = "expr-lang",
      expressionAdapter,
      syntaxProfile = "wrapped",
      expressionMode,
      startWord,
      prefix,
      suffix,
      quickTip,
      ...rest
    } = props;

    const resolvedAdapter = useMemo(
      () => expressionAdapter ?? getExpressionDialectAdapter(dialect) ?? exprLangAdapter,
      [dialect, expressionAdapter],
    );

    const profile = PROFILE[syntaxProfile];

    return (
      <AutoCompleteInput
        ref={forwardedRef}
        expressionAdapter={resolvedAdapter}
        expressionMode={expressionMode ?? profile.expressionMode}
        startWord={startWord ?? profile.startWord}
        prefix={prefix ?? profile.prefix}
        suffix={suffix ?? profile.suffix}
        quickTip={quickTip ?? QUICK_TIP[dialect][syntaxProfile]}
        {...rest}
      />
    );
  },
);
