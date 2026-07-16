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
      includeTopLevelGlobals,
      includeFunctions,
      excludedSuggestions,
      ...rest
    } = props;

    const resolvedAdapter = useMemo(
      () => expressionAdapter ?? getExpressionDialectAdapter(dialect) ?? exprLangAdapter,
      [dialect, expressionAdapter],
    );

    const profile = PROFILE[syntaxProfile];
    // Widget CEL fields address the row context directly (`payload.foo`), so
    // include plain-name completions in every CEL suggestion list.
    const resolvedIncludeTopLevelGlobals = includeTopLevelGlobals ?? dialect === "cel";
    // Only `pathOrRaw` fields expect suggestions to fire outside `{{ … }}` —
    // wrapped-only fields (markdown/HTML title & body) treat prose as literal
    // text and must not open the suggestion dropdown there.
    const pathModeOutsideWrapper = syntaxProfile === "pathOrRaw";
    // The built-in function catalog and the `memory` namespace are expr-lang
    // specific — hide them from CEL fields so authors don't insert identifiers
    // the CEL runtime doesn't understand.
    const resolvedIncludeFunctions = includeFunctions ?? dialect !== "cel";
    // Widget CEL's `$` selector maps to the internal `__runNodes__` map on the
    // row, so route env-key completion (`$` / `$["…"]`) to node names when the
    // caller's `exampleObj` actually carries that map (widget forms). Markdown
    // and HTML editors reuse the CEL dialect but pass a variable dictionary
    // instead, so we leave `envKeySource` unset and let `$` fall back to the
    // top-level globals in those cases.
    const envKeySource = dialect === "cel" && hasRunNodesMap(rest.exampleObj) ? "__runNodes__" : undefined;
    const resolvedExcludedSuggestions = useMemo(() => {
      if (dialect !== "cel") return excludedSuggestions;
      const base = excludedSuggestions ?? [];
      return base.includes("memory") ? base : [...base, "memory"];
    }, [dialect, excludedSuggestions]);

    return (
      <AutoCompleteInput
        ref={forwardedRef}
        expressionAdapter={resolvedAdapter}
        expressionMode={expressionMode ?? profile.expressionMode}
        startWord={startWord ?? profile.startWord}
        prefix={prefix ?? profile.prefix}
        suffix={suffix ?? profile.suffix}
        quickTip={quickTip ?? QUICK_TIP[dialect][syntaxProfile]}
        includeTopLevelGlobals={resolvedIncludeTopLevelGlobals}
        includeFunctions={resolvedIncludeFunctions}
        pathModeOutsideWrapper={pathModeOutsideWrapper}
        envKeySource={envKeySource}
        excludedSuggestions={resolvedExcludedSuggestions}
        {...rest}
      />
    );
  },
);

function hasRunNodesMap(exampleObj: unknown): boolean {
  if (!exampleObj || typeof exampleObj !== "object") return false;
  const record = exampleObj as Record<string, unknown>;
  const map = record.__runNodes__;
  return !!map && typeof map === "object";
}
