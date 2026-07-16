import React, { useState, useEffect, useRef, forwardRef, useImperativeHandle, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { twMerge } from "tailwind-merge";
import type { Suggestion } from "./core";
import { getSuggestions } from "./core";
import { Eye, EyeOff } from "lucide-react";
import type { ExpressionAdapter } from "@/lib/expression";
import { exprLangAdapter } from "@/lib/expression";
import { calculateDropdownPosition } from "./dropdownPosition";

export interface AutoCompleteInputProps extends Omit<React.ComponentPropsWithoutRef<"textarea">, "onChange" | "size"> {
  exampleObj: Record<string, unknown> | null;
  value?: string;
  onChange?: (value: string) => void;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
  prefix?: string;
  suffix?: string;
  startWord?: string;
  inputSize?: "xs" | "sm" | "md" | "lg";
  noExampleObjectText?: string;
  showValuePreview?: boolean;
  valuePreviewLabel?: string;
  quickTip?: string;
  expressionMode?: "wrapped" | "raw";
  /** Labels of suggestions to hide (e.g., ["$", "previous"] to restrict to root() only). */
  excludedSuggestions?: string[];
  /** Minimum height in pixels. Overrides the default derived from `inputSize` (useful for multi-line fields). */
  minHeight?: number;
  /**
   * When true, the input stretches to fill the height of its parent instead of auto-resizing to fit its content.
   * Use this inside modals or panels that provide their own scroll container.
   */
  fullHeight?: boolean;
  /** Adapter that powers evaluation and previews. Defaults to expr-lang. */
  expressionAdapter?: ExpressionAdapter;
  /**
   * Suggest top-level `exampleObj` keys (e.g. `payload`, `status`) as plain-name
   * completions. Enable for dialects like widget CEL where the row context is
   * addressed directly rather than through the `$` selector.
   */
  includeTopLevelGlobals?: boolean;
  /**
   * Include the built-in expr-lang function list (`root()`, `previous()`,
   * `upper()`, …) in suggestions. Disable for dialects with a different
   * function set (e.g. widget CEL).
   */
  includeFunctions?: boolean;
  /**
   * In wrapped mode, also fire suggestions when the cursor sits outside a
   * `{{ … }}` block by treating the whole input as a plain-path expression.
   * Enable for fields that accept either a plain path or a wrapped expression
   * (e.g. widget CEL `pathOrRaw` profile).
   */
  pathModeOutsideWrapper?: boolean;
  /**
   * Key in `exampleObj` that backs the `$` / `$["…"]` selector. Defaults to the
   * globals root (expr-lang). Widget CEL points this at `__runNodes__` so `$`
   * completes run-node names instead of row fields.
   */
  envKeySource?: string;
}

const suggestionSortPriority = {
  $: 1,
  root: 2,
  previous: 3,
  memory: 4,
} as const;

function getActiveStickyHeaderHeight(container: HTMLElement, item: HTMLElement) {
  const containerRect = container.getBoundingClientRect();
  const containerTop = containerRect.top;
  const headers = container.querySelectorAll("[data-suggestion-section-header]");

  let stackedHeight = 0;
  for (const header of headers) {
    if (!(header.compareDocumentPosition(item) & Node.DOCUMENT_POSITION_FOLLOWING)) {
      continue;
    }

    const headerRect = header.getBoundingClientRect();
    const stackTop = containerTop + stackedHeight;
    const isStuck = headerRect.top <= stackTop + 1 && headerRect.bottom > stackTop;
    if (isStuck) {
      stackedHeight += headerRect.height;
    }
  }

  return stackedHeight;
}

function getSuggestionListKey(suggestions: Suggestion[]) {
  return suggestions
    .map((suggestion) =>
      [
        suggestion.kind,
        suggestion.label,
        suggestion.insertText ?? "",
        suggestion.nodeName ?? "",
        suggestion.nodeId ?? "",
      ].join("\u001f"),
    )
    .join("\u001e");
}

// Keep in sync with suggestion section header (`h-7`) and item row (`h-9`) classes.
const SUGGESTION_SECTION_HEADER_HEIGHT_PX = 28;
const SUGGESTION_ITEM_HEIGHT_PX = 36;
const SUGGESTION_LIST_VISIBLE_ITEM_COUNT = 6;
const SUGGESTION_LIST_MAX_HEIGHT_PX =
  SUGGESTION_SECTION_HEADER_HEIGHT_PX + SUGGESTION_ITEM_HEIGHT_PX * SUGGESTION_LIST_VISIBLE_ITEM_COUNT;

const INPUT_SIZE_TYPOGRAPHY = {
  xs: "px-2 py-1 text-xs leading-[18px]",
  sm: "px-2 py-1 text-sm leading-[22px]",
  md: "px-3 py-1 text-sm leading-[22px]",
  lg: "px-4 py-2 text-lg leading-[26px]",
} as const;

const INPUT_SIZE_HEIGHT = {
  xs: "h-7",
  sm: "h-8",
  md: "h-8",
  lg: "h-11",
} as const;

const INPUT_SIZE_MIN_HEIGHT: Record<NonNullable<AutoCompleteInputProps["inputSize"]>, number> = {
  xs: 28,
  sm: 32,
  md: 32,
  lg: 44,
};

export const AutoCompleteInput = forwardRef<HTMLTextAreaElement, AutoCompleteInputProps>(
  function AutoCompleteInputRender(props, forwardedRef) {
    const {
      exampleObj,
      value = "",
      onChange,
      className,
      placeholder = "Type to search...",
      disabled,
      prefix = "",
      suffix = "",
      startWord,
      inputSize = "md",
      noExampleObjectText = "No suggestions found",
      showValuePreview = false,
      valuePreviewLabel = "Preview",
      quickTip,
      expressionMode = "wrapped",
      excludedSuggestions,
      minHeight,
      fullHeight = false,
      expressionAdapter = exprLangAdapter,
      includeTopLevelGlobals = false,
      includeFunctions = true,
      pathModeOutsideWrapper = false,
      envKeySource,
      ...rest
    } = props;
    const [inputValue, setInputValue] = useState(value);
    const [suggestions, setSuggestions] = useState<Array<ReturnType<typeof getSuggestions>[number]>>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    const [highlightedIndex, setHighlightedIndex] = useState(-1);
    const [highlightedValue, setHighlightedValue] = useState<unknown>(undefined);
    const [highlightedSuggestion, setHighlightedSuggestion] = useState<Suggestion | null>(null);
    const [cursorPosition, setCursorPosition] = useState(0);
    const [dropdownPosition, setDropdownPosition] = useState<{
      top: number;
      left: number;
    }>({ top: 0, left: 0 });
    const [previewMode, setPreviewMode] = useState(false);
    const dropdownWidth = 350;
    const previousWordLength = useRef<number>(0);
    const previousInputValue = useRef<string>(value);
    const highlightedIndexRef = useRef(highlightedIndex);
    const suggestionListKeyRef = useRef("");
    const suggestionItemsRef = useRef<Array<ReturnType<typeof getSuggestions>[number]>>([]);

    const isRawExpression = expressionMode === "raw";
    const hasTemplateSegments = /\{\{[\s\S]*?\}\}/.test(inputValue);
    // Preview the whole input as a single value when either:
    //  - raw mode (the whole input IS the expression), or
    //  - `pathOrRaw` mode without `{{ … }}` AND the adapter knows how to
    //    resolve bare input as a literal path (mirroring runtime semantics).
    const canEvaluatePathLiteral =
      pathModeOutsideWrapper && !hasTemplateSegments && typeof expressionAdapter.evaluatePathLiteral === "function";
    const evaluateWholeInputAsExpression = isRawExpression || canEvaluatePathLiteral;

    const evaluateWholeInput = useCallback(
      (input: string, globals: Record<string, unknown>) => {
        if (canEvaluatePathLiteral && expressionAdapter.evaluatePathLiteral) {
          const outcome = expressionAdapter.evaluatePathLiteral(input, globals);
          if (outcome) return outcome;
        }
        return expressionAdapter.evaluate(input, globals);
      },
      [canEvaluatePathLiteral, expressionAdapter],
    );

    const allExpressionsValid = useMemo(() => {
      if (!exampleObj) return true;
      if (evaluateWholeInputAsExpression) {
        if (inputValue.trim().length === 0) return true;
        return evaluateWholeInput(inputValue, exampleObj).ok;
      }
      const regex = /\{\{(.*?)\}\}/g;
      let match;
      while ((match = regex.exec(inputValue)) !== null) {
        if (!expressionAdapter.evaluate(match[1], exampleObj).ok) {
          return false;
        }
      }
      return true;
    }, [inputValue, exampleObj, evaluateWholeInputAsExpression, evaluateWholeInput, expressionAdapter]);

    const containerRef = useRef<HTMLDivElement>(null);
    const suggestionsRef = useRef<HTMLDivElement>(null);
    const suggestionsListRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLTextAreaElement>(null);
    const backdropRef = useRef<HTMLDivElement>(null);
    const mirrorRef = useRef<HTMLDivElement>(null);
    const isInteractingWithSuggestionsRef = useRef(false);
    const suppressSuggestionsRef = useRef(false);
    useImperativeHandle(forwardedRef, () => inputRef.current as HTMLTextAreaElement);

    // Auto-resize textarea based on content (and backdrop in preview mode).
    // In fullHeight mode the parent controls the height, so we skip the auto-resize
    // and clear any inline height that a previous non-fullHeight render might have set.
    const adjustTextareaHeight = useCallback(() => {
      const textarea = inputRef.current;
      const backdrop = backdropRef.current;
      if (!textarea) return;
      if (fullHeight) {
        textarea.style.height = "";
        return;
      }
      textarea.style.height = "auto";
      // In preview mode, backdrop content may be longer than textarea content
      // Use the larger of the two heights
      const textareaHeight = textarea.scrollHeight;
      const backdropHeight = backdrop?.scrollHeight ?? 0;
      const resolvedMinHeight = minHeight ?? INPUT_SIZE_MIN_HEIGHT[inputSize];
      const finalHeight = Math.max(textareaHeight, backdropHeight, resolvedMinHeight);
      textarea.style.height = `${finalHeight}px`;
    }, [fullHeight, inputSize, minHeight]);

    // Tokenize expression content for syntax highlighting
    const tokenizeExpression = (expr: string): React.ReactNode[] => {
      const tokens: React.ReactNode[] = [];
      let i = 0;
      let key = 0;

      while (i < expr.length) {
        // Skip whitespace
        if (/\s/.test(expr[i])) {
          let ws = "";
          while (i < expr.length && /\s/.test(expr[i])) {
            ws += expr[i++];
          }
          tokens.push(<span key={key++}>{ws}</span>);
          continue;
        }

        // $ selector (root)
        if (expr[i] === "$") {
          tokens.push(
            <span key={key++} className="text-violet-600 dark:text-violet-400 font-semibold">
              $
            </span>,
          );
          i++;
          continue;
        }

        // Property access with dot notation
        if (expr[i] === ".") {
          tokens.push(
            <span key={key++} className="text-gray-500 dark:text-gray-400">
              .
            </span>,
          );
          i++;
          // Capture the property name
          let prop = "";
          while (i < expr.length && /[a-zA-Z0-9_]/.test(expr[i])) {
            prop += expr[i++];
          }
          if (prop) {
            tokens.push(
              <span key={key++} className="text-sky-600 dark:text-sky-400">
                {prop}
              </span>,
            );
          }
          continue;
        }

        // Strings (single or double quoted)
        if (expr[i] === '"' || expr[i] === "'") {
          const quote = expr[i];
          let str = quote;
          i++;
          while (i < expr.length && expr[i] !== quote) {
            if (expr[i] === "\\" && i + 1 < expr.length) {
              str += expr[i++];
            }
            str += expr[i++];
          }
          if (i < expr.length) str += expr[i++]; // closing quote
          tokens.push(
            <span key={key++} className="text-amber-600 dark:text-amber-400">
              {str}
            </span>,
          );
          continue;
        }

        // Numbers
        if (/[0-9]/.test(expr[i])) {
          let num = "";
          while (i < expr.length && /[0-9.]/.test(expr[i])) {
            num += expr[i++];
          }
          tokens.push(
            <span key={key++} className="text-orange-600 dark:text-orange-400">
              {num}
            </span>,
          );
          continue;
        }

        // Identifiers (could be functions, keywords, or node names)
        if (/[a-zA-Z_]/.test(expr[i])) {
          let ident = "";
          while (i < expr.length && /[a-zA-Z0-9_]/.test(expr[i])) {
            ident += expr[i++];
          }
          // Check if it's a function call (followed by parenthesis)
          const isFunction = i < expr.length && expr[i] === "(";
          // Check if it's a keyword
          const keywords = [
            "true",
            "false",
            "nil",
            "null",
            "in",
            "not",
            "and",
            "or",
            "matches",
            "contains",
            "startsWith",
            "endsWith",
          ];
          const isKeyword = keywords.includes(ident);

          if (isFunction) {
            tokens.push(
              <span key={key++} className="text-emerald-600 dark:text-emerald-400">
                {ident}
              </span>,
            );
          } else if (isKeyword) {
            tokens.push(
              <span key={key++} className="text-pink-600 dark:text-pink-400 font-medium">
                {ident}
              </span>,
            );
          } else {
            // Regular identifier (likely a node name or variable)
            tokens.push(
              <span key={key++} className="text-sky-600 dark:text-sky-400">
                {ident}
              </span>,
            );
          }
          continue;
        }

        // Operators and punctuation
        const operators = [
          "==",
          "!=",
          ">=",
          "<=",
          "&&",
          "||",
          "??",
          "?:",
          "->",
          "..",
          ">",
          "<",
          "+",
          "-",
          "*",
          "/",
          "%",
          "!",
          "?",
          ":",
          "[",
          "]",
          "(",
          ")",
          ",",
          "|",
        ];
        let foundOp = false;
        for (const op of operators) {
          if (expr.slice(i, i + op.length) === op) {
            tokens.push(
              <span key={key++} className="text-gray-500 dark:text-gray-400">
                {op}
              </span>,
            );
            i += op.length;
            foundOp = true;
            break;
          }
        }
        if (foundOp) continue;

        // Fallback: single character
        tokens.push(<span key={key++}>{expr[i++]}</span>);
      }

      return tokens;
    };

    // Evaluate a `{{ … }}` inner segment against the exampleObj.
    const evaluateExpression = useCallback(
      (expr: string): { value: string; error?: string } => {
        if (!exampleObj) return { value: "?", error: "No context available" };
        const outcome = expressionAdapter.evaluate(expr, exampleObj);
        if (!outcome.ok) {
          return { value: "error", error: outcome.error ?? "Evaluation failed" };
        }
        return { value: outcome.formattedValue ?? expressionAdapter.formatResult(outcome.value) };
      },
      [exampleObj, expressionAdapter],
    );

    // Evaluate the whole input (raw mode / `pathOrRaw` plain input).
    const evaluateWholeInputPreview = useCallback(
      (input: string): { value: string; error?: string } => {
        if (!exampleObj) return { value: "?", error: "No context available" };
        const outcome = evaluateWholeInput(input, exampleObj);
        if (!outcome.ok) {
          return { value: "error", error: outcome.error ?? "Evaluation failed" };
        }
        return { value: outcome.formattedValue ?? expressionAdapter.formatResult(outcome.value) };
      },
      [exampleObj, evaluateWholeInput, expressionAdapter],
    );

    // Render content with highlighted expressions
    const renderHighlightedContent = (text: string) => {
      // Raw mode + `pathOrRaw` fields with no `{{ … }}` share the same preview
      // treatment: the whole input is one expression to evaluate.
      if (evaluateWholeInputAsExpression) {
        if (!text) {
          return [<span key={0}>{"\u200B"}</span>];
        }
        if (previewMode) {
          const result = evaluateWholeInputPreview(text);
          if (result.error) {
            return [
              <span key={0} className="bg-red-100 dark:bg-red-900/50 rounded-sm">
                <span className="text-red-600 dark:text-red-400 font-medium">{` error (${result.error}) `}</span>
              </span>,
            ];
          }
          return [
            <span key={0} className="bg-emerald-100 dark:bg-emerald-900/50 rounded-sm">
              <span className="text-emerald-700 dark:text-emerald-300 font-medium">{` ${result.value} `}</span>
            </span>,
          ];
        }
        return tokenizeExpression(text);
      }

      const parts: React.ReactNode[] = [];
      const regex = /(\{\{)(.*?)(\}\})/g;
      let lastIndex = 0;
      let match;
      let key = 0;

      while ((match = regex.exec(text)) !== null) {
        // Add text before the match
        if (match.index > lastIndex) {
          parts.push(<span key={key++}>{text.slice(lastIndex, match.index)}</span>);
        }
        // Add the highlighted expression with syntax coloring
        if (previewMode) {
          // In preview mode, show evaluated value or error
          const result = evaluateExpression(match[2]);
          if (result.error) {
            // Error state - show in red with error message
            parts.push(
              <span key={key++} className="bg-red-100 dark:bg-red-900/50 rounded-sm">
                <span className="text-gray-400 dark:text-gray-500">{match[1]}</span>
                <span className="text-red-600 dark:text-red-400 font-medium">{` error (${result.error}) `}</span>
                <span className="text-gray-400 dark:text-gray-500">{match[3]}</span>
              </span>,
            );
          } else {
            // Success state - show in green
            parts.push(
              <span key={key++} className="bg-emerald-100 dark:bg-emerald-900/50 rounded-sm">
                <span className="text-gray-400 dark:text-gray-500">{match[1]}</span>
                <span className="text-emerald-700 dark:text-emerald-300 font-medium">{` ${result.value} `}</span>
                <span className="text-gray-400 dark:text-gray-500">{match[3]}</span>
              </span>,
            );
          }
        } else {
          parts.push(
            <span key={key++} className="bg-slate-100 dark:bg-gray-800 rounded-sm">
              <span className="text-gray-400 dark:text-gray-500">{match[1]}</span>
              {tokenizeExpression(match[2])}
              <span className="text-gray-400 dark:text-gray-500">{match[3]}</span>
            </span>,
          );
        }
        lastIndex = regex.lastIndex;
      }

      // Add remaining text
      if (lastIndex < text.length) {
        parts.push(<span key={key++}>{text.slice(lastIndex)}</span>);
      }

      // Handle empty text - add a zero-width space to maintain height
      if (parts.length === 0) {
        parts.push(<span key={0}>{"\u200B"}</span>);
      }

      return parts;
    };

    const getWordAtCursor = (text: string, position: number) => {
      const beforeCursor = text.substring(0, position);
      const afterCursor = text.substring(position);

      const wordStart = Math.max(0, beforeCursor.lastIndexOf(" ") + 1);
      const wordEndInAfter = afterCursor.indexOf(" ");
      const wordEnd = wordEndInAfter === -1 ? text.length : position + wordEndInAfter;

      const word = text.substring(wordStart, wordEnd);
      return {
        word,
        start: wordStart,
        end: wordEnd,
      };
    };

    const isAllowedToSuggest = useCallback(
      (text: string, position: number) => {
        if (isRawExpression || !props.startWord || !props.suffix) {
          return true;
        }

        const openIndex = text.lastIndexOf(props.startWord, position);
        if (openIndex === -1) {
          return pathModeOutsideWrapper;
        }

        const closeIndex = text.indexOf(props.suffix, openIndex + 2);
        if (closeIndex !== -1 && position - 1 > closeIndex) {
          return pathModeOutsideWrapper;
        }

        return true;
      },
      [isRawExpression, props.startWord, props.suffix, pathModeOutsideWrapper],
    );

    const getExpressionContext = useCallback(
      (text: string, cursor: number) => {
        if (isRawExpression || !startWord || !suffix) {
          return {
            expressionText: text,
            expressionCursor: cursor,
            startOffset: 0,
            endOffset: text.length,
          };
        }

        const openIndex = text.lastIndexOf(startWord, cursor);
        const closeIndex = openIndex === -1 ? -1 : text.indexOf(suffix, openIndex + startWord.length);
        const insideWrapper = openIndex !== -1 && (closeIndex === -1 || cursor - 1 <= closeIndex);

        if (!insideWrapper) {
          // Path-mode fallback: treat the whole input as a plain-path
          // expression when the field opts in (e.g. widget CEL `pathOrRaw`).
          if (!pathModeOutsideWrapper) return null;
          return {
            expressionText: text,
            expressionCursor: cursor,
            startOffset: 0,
            endOffset: text.length,
          };
        }

        const startOffset = openIndex + startWord.length;
        const endOffset = closeIndex === -1 ? text.length : closeIndex;
        return {
          expressionText: text.slice(startOffset, endOffset),
          expressionCursor: Math.max(0, cursor - startOffset),
          startOffset,
          endOffset,
        };
      },
      [isRawExpression, startWord, suffix, pathModeOutsideWrapper],
    );

    const getSuggestionInsertText = (suggestion: ReturnType<typeof getSuggestions>[number]) => {
      if (suggestion.kind === "function") {
        if (suggestion.label === "root" || suggestion.label === "previous") {
          return suggestion.insertText ?? `${suggestion.label}().`;
        }
        return `${suggestion.label}()`;
      }
      return suggestion.insertText ?? suggestion.label;
    };

    const formatFunctionSignature = (suggestion: ReturnType<typeof getSuggestions>[number]) => {
      if (suggestion.kind !== "function") {
        return "";
      }

      const insertText = suggestion.insertText ?? `${suggestion.label}()`;
      const openParen = insertText.indexOf("(");
      const closeParen = insertText.lastIndexOf(")");
      if (openParen === -1 || closeParen === -1 || closeParen <= openParen) {
        return "()";
      }

      let params = insertText.slice(openParen + 1, closeParen);
      params = params.replace(/\$\{\d+:([^}]+)\}/g, "$1");
      params = params.replace(/\$\{\d+\}/g, "");
      params = params.replace(/\$0/g, "");
      params = params.replace(/\s+/g, " ").trim();
      params = params.replace(/\s+,/g, ",").replace(/,\s+/g, ", ");

      return `(${params})`;
    };

    const getSuggestionDisplayLabel = (suggestion: Suggestion) => {
      if (suggestion.kind === "function" && (suggestion.label === "root" || suggestion.label === "previous")) {
        return `${suggestion.label}()`;
      }

      return suggestion.label;
    };

    const getSortedSuggestions = useCallback(
      (expressionText: string, expressionCursor: number) =>
        getSuggestions(expressionText, expressionCursor, exampleObj ?? {}, {
          limit: 150,
          includeTopLevelGlobals,
          includeFunctions,
          envKeySource,
        })
          .filter((s) => !excludedSuggestions?.includes(s.label))
          .sort((a, b) => {
            const aPriority = suggestionSortPriority[a.label as keyof typeof suggestionSortPriority];
            const bPriority = suggestionSortPriority[b.label as keyof typeof suggestionSortPriority];

            if (aPriority !== undefined && bPriority !== undefined) {
              return aPriority - bPriority;
            }
            if (aPriority !== undefined) {
              return -1;
            }
            if (bPriority !== undefined) {
              return 1;
            }
            return a.label.localeCompare(b.label);
          }),
      [exampleObj, excludedSuggestions, includeTopLevelGlobals, includeFunctions, envKeySource],
    );

    const getReplacementRange = (left: string, insertText: string) => {
      const envBracketMatch = left.match(/\$env\s*\[\s*(['"])([^'"]*)$/);
      if (envBracketMatch) {
        const partial = envBracketMatch[2] ?? "";
        return { start: left.length - (partial.length + 1), end: left.length };
      }

      const dollarBracketMatch = left.match(/\$\s*\[\s*(['"])([^'"]*)$/);
      if (dollarBracketMatch) {
        const partial = dollarBracketMatch[2] ?? "";
        return { start: left.length - (partial.length + 1), end: left.length };
      }

      const envTriggerMatch = left.match(/\$env\s*\[\s*$/);
      if (envTriggerMatch && envTriggerMatch.index !== undefined) {
        return { start: envTriggerMatch.index, end: left.length };
      }

      const dollarTriggerMatch = left.match(/\$\s*\[\s*$|\$\s*$/);
      if (dollarTriggerMatch && dollarTriggerMatch.index !== undefined) {
        return { start: dollarTriggerMatch.index, end: left.length };
      }

      const dotMatch = left.match(/(.+?)\.\s*([$A-Za-z_][$A-Za-z0-9_]*)?$/);
      if (dotMatch) {
        const memberPrefix = dotMatch[2] ?? "";
        let start = left.length - memberPrefix.length;
        if (insertText.startsWith("[") && left[start - 1] === ".") {
          start -= 1;
        }
        return { start, end: left.length };
      }

      const identMatch = left.match(/[$A-Za-z_][$A-Za-z0-9_]*$/);
      if (identMatch) {
        return { start: left.length - identMatch[0].length, end: left.length };
      }

      return { start: left.length, end: left.length };
    };

    const computeHighlightedValue = React.useCallback(
      (suggestion: ReturnType<typeof getSuggestions>[number], context: ReturnType<typeof getExpressionContext>) => {
        if (!exampleObj || !context) return undefined;
        if (suggestion.kind === "function") return undefined;

        const insertText = getSuggestionInsertText(suggestion);
        const left = context.expressionText.slice(0, context.expressionCursor);
        const range = getReplacementRange(left, insertText);
        const nextExpressionLeft = context.expressionText.slice(0, range.start) + insertText;
        const value = expressionAdapter.resolveSuggestionValue(nextExpressionLeft, exampleObj);
        if (typeof value === "function") return undefined;
        return value;
      },
      [exampleObj, expressionAdapter],
    );

    const valuePreviewWidth = 200;
    const valuePreviewCodeBlockClassName =
      "text-xs font-mono bg-slate-100 dark:bg-gray-800 rounded-lg px-2.5 py-2 text-gray-800 dark:text-gray-200";

    const renderQuickTip = (tip: string) => {
      const parts = tip.split(/`([^`]+)`/g);
      return parts.map((part, index) =>
        index % 2 === 1 ? (
          <code
            key={`code-${index}`}
            className="bg-slate-100 dark:bg-gray-700 px-1 py-0.5 rounded text-gray-700 dark:text-gray-300"
          >
            {part}
          </code>
        ) : (
          <span key={`text-${index}`}>{part}</span>
        ),
      );
    };

    const measureCursorPixelPosition = useCallback(() => {
      if (!inputRef.current || !mirrorRef.current) return;

      const input = inputRef.current;
      const mirror = mirrorRef.current;
      const computed = getComputedStyle(input);
      const inputRect = input.getBoundingClientRect();
      const containerRect = containerRef.current?.getBoundingClientRect();

      mirror.style.font = computed.font;
      mirror.style.fontSize = computed.fontSize;
      mirror.style.fontFamily = computed.fontFamily;
      mirror.style.fontWeight = computed.fontWeight;
      mirror.style.letterSpacing = computed.letterSpacing;
      mirror.style.textTransform = computed.textTransform;
      mirror.style.lineHeight = computed.lineHeight;
      mirror.style.padding = computed.padding;
      mirror.style.border = computed.border;
      mirror.style.boxSizing = computed.boxSizing;
      mirror.style.width = `${inputRect.width}px`;
      mirror.style.top = containerRect ? `${inputRect.top - containerRect.top}px` : "0";
      mirror.style.left = containerRect ? `${inputRect.left - containerRect.left}px` : "0";
      mirror.style.whiteSpace = "pre-wrap";
      mirror.style.overflowWrap = "break-word";
      mirror.style.wordBreak = computed.wordBreak;
      mirror.style.overflow = "hidden";

      mirror.replaceChildren(document.createTextNode(inputValue.substring(0, cursorPosition)));

      const cursorMarker = document.createElement("span");
      cursorMarker.textContent = "\u200b";
      mirror.appendChild(cursorMarker);

      const markerRect = cursorMarker.getBoundingClientRect();
      const cursor = {
        x: markerRect.left - input.scrollLeft,
        y: markerRect.bottom - input.scrollTop,
      };

      setDropdownPosition(
        calculateDropdownPosition({
          cursor,
          viewportWidth: window.innerWidth,
          dropdownWidth,
          valuePreviewWidth,
          showValuePreview,
        }),
      );
    }, [inputValue, cursorPosition, dropdownWidth, showValuePreview]);

    // Measure cursor pixel position when cursor or input changes
    useEffect(() => {
      measureCursorPixelPosition();
    }, [measureCursorPixelPosition]);

    // Sync scroll between textarea and backdrop, then keep the fixed-position
    // suggestions portal anchored to the scrolled caret.
    const handleScroll = useCallback(() => {
      if (inputRef.current && backdropRef.current) {
        backdropRef.current.scrollTop = inputRef.current.scrollTop;
        backdropRef.current.scrollLeft = inputRef.current.scrollLeft;
      }

      measureCursorPixelPosition();
    }, [measureCursorPixelPosition]);

    // Keep the fixed-position dropdown anchored to the input while any ancestor
    // scrolls or the window resizes. Without this the dropdown detaches from the
    // input when the surrounding panel scrolls (issue #3615).
    useEffect(() => {
      if (!isOpen || suggestions.length === 0) {
        return;
      }

      let frame = 0;
      const reposition = () => {
        if (frame) {
          return;
        }
        frame = requestAnimationFrame(() => {
          frame = 0;
          measureCursorPixelPosition();
        });
      };

      // Capture phase so scrolling of any scrollable ancestor is observed, not just window.
      window.addEventListener("scroll", reposition, true);
      window.addEventListener("resize", reposition);
      return () => {
        if (frame) {
          cancelAnimationFrame(frame);
        }
        window.removeEventListener("scroll", reposition, true);
        window.removeEventListener("resize", reposition);
      };
    }, [isOpen, suggestions.length, measureCursorPixelPosition]);

    useEffect(() => {
      setInputValue(value);
    }, [value]);

    // Adjust textarea height when value or preview mode changes
    useEffect(() => {
      adjustTextareaHeight();
    }, [inputValue, previewMode, adjustTextareaHeight]);

    useEffect(() => {
      previousInputValue.current = inputValue;
    }, [inputValue]);

    useEffect(() => {
      highlightedIndexRef.current = highlightedIndex;
    }, [highlightedIndex]);

    useEffect(() => {
      suggestionItemsRef.current = suggestions;
    }, [suggestions]);

    useEffect(() => {
      if (!isFocused) {
        suggestionListKeyRef.current = "";
        highlightedIndexRef.current = -1;
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedIndex(-1);
        setHighlightedValue(undefined);
        setHighlightedSuggestion(null);
        return;
      }

      if (suppressSuggestionsRef.current) {
        suppressSuggestionsRef.current = false;
        return;
      }

      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context || !isAllowedToSuggest(inputValue, cursorPosition)) {
        suggestionListKeyRef.current = "";
        highlightedIndexRef.current = -1;
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedIndex(-1);
        setHighlightedValue(undefined);
        setHighlightedSuggestion(null);
        return;
      }

      const newSuggestions = getSortedSuggestions(context.expressionText, context.expressionCursor);
      setIsOpen(newSuggestions.length > 0);

      const suggestionListKey = getSuggestionListKey(newSuggestions);
      const hasSameSuggestionList = suggestionListKeyRef.current === suggestionListKey;
      const canPreserveHighlightedIndex =
        hasSameSuggestionList &&
        highlightedIndexRef.current >= 0 &&
        highlightedIndexRef.current < newSuggestions.length;

      const activeSuggestions = hasSameSuggestionList ? suggestionItemsRef.current : newSuggestions;
      if (!hasSameSuggestionList) {
        setSuggestions(newSuggestions);
      }

      suggestionListKeyRef.current = suggestionListKey;

      const nextHighlightedIndex = canPreserveHighlightedIndex
        ? highlightedIndexRef.current
        : showValuePreview && newSuggestions.length > 0
          ? 0
          : -1;
      highlightedIndexRef.current = nextHighlightedIndex;
      setHighlightedIndex(nextHighlightedIndex);
      if (nextHighlightedIndex >= 0) {
        const suggestion = activeSuggestions[nextHighlightedIndex] ?? newSuggestions[nextHighlightedIndex];
        setHighlightedSuggestion(suggestion);
        const value = computeHighlightedValue(suggestion, context);
        setHighlightedValue(value);
      } else {
        setHighlightedValue(undefined);
        setHighlightedSuggestion(null);
      }
    }, [
      inputValue,
      cursorPosition,
      isFocused,
      startWord,
      suffix,
      onChange,
      showValuePreview,
      exampleObj,
      excludedSuggestions,
      computeHighlightedValue,
      getExpressionContext,
      getSortedSuggestions,
      isAllowedToSuggest,
    ]);

    // Handle clicking outside to close suggestions
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (suggestionsRef.current?.contains(event.target as Node)) {
          return;
        }
        if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
          setIsOpen(false);
          setIsFocused(false);
          inputRef.current?.blur();
          setHighlightedIndex(-1);
          setHighlightedValue(undefined);
        }
      };

      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }, []);

    const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newValue = e.target.value;
      const cursorPosition = e.target.selectionStart ?? newValue.length;
      setCursorPosition(cursorPosition);
      const { word, start } = getWordAtCursor(newValue, cursorPosition);
      const beforeCursor = newValue.slice(0, cursorPosition);
      const afterCursor = newValue.slice(cursorPosition);
      const priorValue = previousInputValue.current;
      const wasSingleCharInsert = newValue.length === priorValue.length + 1;
      const insertedChar = wasSingleCharInsert ? newValue[cursorPosition - 1] : "";
      const isInsertAtCursor = wasSingleCharInsert && priorValue === `${beforeCursor.slice(0, -1)}${afterCursor}`;

      if (
        startWord &&
        word === startWord &&
        previousWordLength.current < word.length &&
        insertedChar === "{" &&
        isInsertAtCursor &&
        beforeCursor.endsWith(startWord) &&
        !afterCursor.startsWith("}") &&
        !isAllowedToSuggest(inputValue, cursorPosition)
      ) {
        const composedValue = `${newValue.slice(0, start)}${prefix || ""}${suffix || ""}${newValue.slice(start + word.length)}`;
        setInputValue(composedValue);
        onChange?.(composedValue);
        setSuggestions([]);
        setIsOpen(false);
        requestAnimationFrame(() => {
          const cursorTarget = start + (prefix || "").length;
          inputRef.current?.setSelectionRange(cursorTarget, cursorTarget);
        });
        return;
      }

      setInputValue(newValue);
      onChange?.(newValue);
      previousWordLength.current = word.length;
    };

    const handleCursorChange = () => {
      requestAnimationFrame(() => {
        const nextCursorPosition = inputRef.current?.selectionStart ?? inputValue.length;
        setCursorPosition(nextCursorPosition);
      });
    };

    const isKeyboardSuggestionCommit = (key: string) =>
      (key === "Enter" || key === "Tab") &&
      isOpen &&
      highlightedIndexRef.current >= 0 &&
      highlightedIndexRef.current < suggestions.length;

    const handleKeyDownCapture = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (isKeyboardSuggestionCommit(e.key)) {
        return;
      }

      handleCursorChange();
    };

    const showFollowUpSuggestions = useCallback(
      (
        nextSuggestions: Array<ReturnType<typeof getSuggestions>[number]>,
        nextContext: NonNullable<ReturnType<typeof getExpressionContext>>,
      ) => {
        suppressSuggestionsRef.current = false;
        suggestionItemsRef.current = nextSuggestions;
        suggestionListKeyRef.current = getSuggestionListKey(nextSuggestions);
        setSuggestions(nextSuggestions);
        setIsOpen(true);

        const nextHighlightedIndex = showValuePreview ? 0 : -1;
        highlightedIndexRef.current = nextHighlightedIndex;
        setHighlightedIndex(nextHighlightedIndex);

        if (nextHighlightedIndex >= 0) {
          const nextSuggestion = nextSuggestions[nextHighlightedIndex];
          setHighlightedSuggestion(nextSuggestion);
          setHighlightedValue(computeHighlightedValue(nextSuggestion, nextContext));
        } else {
          setHighlightedSuggestion(null);
          setHighlightedValue(undefined);
        }
      },
      [computeHighlightedValue, showValuePreview],
    );

    const closeSuggestions = useCallback(() => {
      highlightedIndexRef.current = -1;
      setHighlightedIndex(-1);
      setHighlightedSuggestion(null);
      setHighlightedValue(undefined);
      setIsOpen(false);
    }, []);

    const handleSuggestionClick = (suggestionItem: ReturnType<typeof getSuggestions>[number]) => {
      suppressSuggestionsRef.current = true;
      const cursorPosition = inputRef.current?.selectionStart || 0;
      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context) {
        suppressSuggestionsRef.current = false;
        isInteractingWithSuggestionsRef.current = false;
        closeSuggestions();
        return;
      }

      const left = context.expressionText.slice(0, context.expressionCursor);
      const insertText = getSuggestionInsertText(suggestionItem);
      const range = getReplacementRange(left, insertText);
      const nextExpression =
        context.expressionText.slice(0, range.start) + insertText + context.expressionText.slice(range.end);
      const newValue = inputValue.slice(0, context.startOffset) + nextExpression + inputValue.slice(context.endOffset);
      const cursorTarget = context.startOffset + range.start + insertText.length;
      const nextContext = getExpressionContext(newValue, cursorTarget);
      const nextSuggestions =
        nextContext && isAllowedToSuggest(newValue, cursorTarget)
          ? getSortedSuggestions(nextContext.expressionText, nextContext.expressionCursor)
          : [];

      setInputValue(newValue);
      onChange?.(newValue);
      setCursorPosition(cursorTarget);

      if (nextContext && nextSuggestions.length > 0) {
        showFollowUpSuggestions(nextSuggestions, nextContext);
      } else {
        closeSuggestions();
      }

      requestAnimationFrame(() => {
        inputRef.current?.focus();
        inputRef.current?.setSelectionRange(cursorTarget, cursorTarget);
        isInteractingWithSuggestionsRef.current = false;
        suppressSuggestionsRef.current = false;
      });
    };

    const scrollHighlightedSuggestionIntoView = useCallback((index: number) => {
      const container = suggestionsListRef.current;
      if (!container || index < 0) return;

      const highlightedElement = container.querySelector(`[data-suggestion-index="${index}"]`) as HTMLElement | null;
      if (!highlightedElement) return;

      const containerRect = container.getBoundingClientRect();
      const itemRect = highlightedElement.getBoundingClientRect();
      const stickyHeaderHeight = getActiveStickyHeaderHeight(container, highlightedElement);
      const effectiveTop = containerRect.top + stickyHeaderHeight;

      if (itemRect.bottom > containerRect.bottom) {
        container.scrollTop = Math.round(container.scrollTop + itemRect.bottom - containerRect.bottom);
      } else if (itemRect.top < effectiveTop) {
        container.scrollTop = Math.round(container.scrollTop - (effectiveTop - itemRect.top));
      }
    }, []);

    const highlightSuggestionAtIndex = useCallback(
      (index: number) => {
        const suggestion = suggestions[index];
        highlightedIndexRef.current = index;
        setHighlightedIndex(index);

        if (!suggestion) {
          setHighlightedSuggestion(null);
          setHighlightedValue(undefined);
          return;
        }

        setHighlightedSuggestion(suggestion);
        const cursorPosition = inputRef.current?.selectionStart || 0;
        const context = getExpressionContext(inputValue, cursorPosition);
        const value = computeHighlightedValue(suggestion, context);
        setHighlightedValue(value);
      },
      [computeHighlightedValue, getExpressionContext, inputValue, suggestions],
    );

    const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!isOpen || suggestions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          highlightSuggestionAtIndex(Math.min(highlightedIndexRef.current + 1, suggestions.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          highlightSuggestionAtIndex(Math.max(highlightedIndexRef.current - 1, 0));
          break;
        case "Enter":
          if (isKeyboardSuggestionCommit(e.key)) {
            e.preventDefault();
            e.stopPropagation();
            isInteractingWithSuggestionsRef.current = true;
            handleSuggestionClick(suggestions[highlightedIndexRef.current]);
          }
          // Allow default behavior (newline) when no suggestion is highlighted
          break;
        case "Tab":
          if (isKeyboardSuggestionCommit(e.key)) {
            e.preventDefault();
            e.stopPropagation();
            isInteractingWithSuggestionsRef.current = true;
            handleSuggestionClick(suggestions[highlightedIndexRef.current]);
          }
          break;
        case "Escape":
          e.preventDefault();
          e.stopPropagation();
          closeSuggestions();
          break;
      }
    };

    // Scroll highlighted item into view with minimal movement (avoid scroll jumps).
    useEffect(() => {
      scrollHighlightedSuggestionIntoView(highlightedIndex);
    }, [highlightedIndex, scrollHighlightedSuggestionIntoView]);

    // Always show Value Preview box when enabled and something is highlighted
    // to prevent position jumping when switching between suggestion types
    const shouldShowValuePreview = showValuePreview && highlightedIndex >= 0;

    const showPreviewToggle = showValuePreview;
    const showBottomBar = showPreviewToggle || (isFocused && !!quickTip);

    return (
      <div
        ref={containerRef}
        data-autocomplete-input=""
        data-autocomplete-suggestions-open={isOpen && suggestions.length > 0 ? "" : undefined}
        className={twMerge(["relative w-full", fullHeight && "flex h-full flex-col"])}
      >
        {/* Hidden mirror element for measuring cursor position */}
        <div
          ref={mirrorRef}
          aria-hidden="true"
          style={{
            position: "absolute",
            visibility: "hidden",
            whiteSpace: "pre-wrap",
            pointerEvents: "none",
            top: 0,
            left: 0,
          }}
        />
        {/* Input Field with Syntax Highlighting */}
        <span
          data-slot="control"
          className={twMerge([
            "relative block w-full rounded-md bg-white dark:bg-gray-800",
            "focus-within:ring-ring/50",
            "has-data-disabled:opacity-50",
            fullHeight && "flex min-h-0 flex-1",
            className,
          ])}
        >
          {/* Backdrop for syntax highlighting */}
          <div
            ref={backdropRef}
            aria-hidden="true"
            className={twMerge([
              "font-sm pointer-events-none absolute inset-0 whitespace-pre-wrap break-words overflow-hidden",
              "rounded-md border border-transparent text-gray-950 dark:text-white",
              INPUT_SIZE_TYPOGRAPHY[inputSize],
            ])}
          >
            {renderHighlightedContent(inputValue)}
          </div>
          {/* Textarea with transparent text */}
          <textarea
            ref={inputRef}
            rows={1}
            value={inputValue}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onKeyUp={handleCursorChange}
            onKeyDownCapture={handleKeyDownCapture}
            onClick={handleCursorChange}
            onSelect={handleCursorChange}
            onScroll={handleScroll}
            onFocus={() => {
              setIsFocused(true);
              if (suggestions.length > 0) {
                setIsOpen(true);
              }
            }}
            onBlur={() => {
              if (isInteractingWithSuggestionsRef.current) {
                requestAnimationFrame(() => {
                  inputRef.current?.focus();
                  isInteractingWithSuggestionsRef.current = false;
                });
                return;
              }
              // Small delay to allow click on suggestions
              setTimeout(() => {
                setIsFocused(false);
                setIsOpen(false);
                setHighlightedValue(undefined);
              }, 150);
            }}
            placeholder={placeholder}
            disabled={disabled || previewMode}
            className={twMerge([
              "font-sm bg-transparent border-gray-300 placeholder:text-gray-500 dark:border-gray-600/70 dark:placeholder:text-gray-500",
              "relative block w-full min-w-0 appearance-none rounded-md border outline-none resize-none overflow-hidden",
              "focus:border-gray-500 focus:shadow-none focus:ring-0 dark:focus:border-gray-500",
              "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
              "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
              // Make text transparent but keep caret visible
              "text-transparent caret-gray-950 dark:caret-white",
              INPUT_SIZE_TYPOGRAPHY[inputSize],
              INPUT_SIZE_HEIGHT[inputSize],
              fullHeight && "h-full min-h-0 flex-1 overflow-y-auto",
            ])}
            {...rest}
          />
        </span>
        {/* Bottom bar with preview toggle and quickTip — rendered in normal flow so it
            never overlaps the input regardless of the parent layout (grid, flex, etc.) */}
        {showBottomBar && (
          <div className="flex items-center justify-between mt-1 px-0.5">
            {/* Preview toggle - left side */}
            {showPreviewToggle ? (
              <button
                type="button"
                onClick={() => setPreviewMode(!previewMode)}
                className={twMerge([
                  "flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium transition-colors",
                  previewMode
                    ? allExpressionsValid
                      ? "bg-emerald-100 dark:bg-emerald-900/50 text-emerald-700 dark:text-emerald-300"
                      : "bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300"
                    : allExpressionsValid
                      ? "text-emerald-600 dark:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-900/30"
                      : "text-red-500 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30",
                ])}
              >
                {previewMode ? <Eye className="w-3 h-3" /> : <EyeOff className="w-3 h-3" />}
                <span>{valuePreviewLabel}</span>
              </button>
            ) : (
              <span />
            )}
            {/* QuickTip - right side */}
            <span className="flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
              {quickTip
                ? renderQuickTip(quickTip)
                : [
                    "Use ",
                    <code
                      key="default-tip"
                      className="bg-slate-100 dark:bg-gray-700 px-1 py-0.5 rounded text-gray-700 dark:text-gray-300"
                    >
                      {"{{"}
                    </code>,
                    " to write ",
                    <a
                      key="expr-link"
                      href="https://expr-lang.org/docs/language-definition"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 dark:text-blue-400 hover:underline"
                    >
                      expr
                    </a>,
                    " expressions",
                  ]}
            </span>
          </div>
        )}

        {/* Suggestions Dropdown - rendered in portal to escape overflow:hidden */}
        {isOpen &&
          suggestions.length > 0 &&
          createPortal(
            <div
              ref={suggestionsRef}
              data-testid="autocomplete-suggestions"
              data-autocomplete-suggestions=""
              className="fixed z-[9999] bg-transparent"
              style={{
                top: `${dropdownPosition.top}px`,
                left: `${dropdownPosition.left}px`,
              }}
            >
              <div className="flex flex-col sm:flex-row">
                {shouldShowValuePreview && isOpen && (
                  <div
                    data-testid="autocomplete-value-preview"
                    className="border border-gray-200 dark:border-gray-600 sm:border-r-0 sm:border-t p-3 bg-white dark:bg-gray-800 sm:rounded-l-lg sm:rounded-br-none h-fit self-start shadow-lg dark:shadow-gray-950/50"
                    style={{ width: `${valuePreviewWidth}px` }}
                  >
                    {/* $ selector */}
                    {highlightedSuggestion?.label === "$" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">$ (Event Data)</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                          {highlightedSuggestion.description}
                        </div>
                        <div className={twMerge(valuePreviewCodeBlockClassName, "text-sky-700 dark:text-sky-400")}>
                          {highlightedSuggestion.nodeCount ?? 0} node
                          {highlightedSuggestion.nodeCount !== 1 ? "s" : ""} available
                        </div>
                      </>
                    ) : /* Node suggestions */
                    highlightedSuggestion?.nodeName ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">
                          {highlightedSuggestion.componentType || "Component"}
                        </div>
                        {highlightedSuggestion.description && (
                          <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                            {highlightedSuggestion.description}
                          </div>
                        )}
                        <div className={twMerge(valuePreviewCodeBlockClassName, "space-y-1")}>
                          <div className="flex justify-between">
                            <span className="text-gray-600 dark:text-gray-400">Name</span>
                            <span className="text-sky-700 dark:text-sky-400 truncate ml-2 max-w-[120px]">
                              {highlightedSuggestion.nodeName}
                            </span>
                          </div>
                          {highlightedSuggestion.nodeId &&
                            highlightedSuggestion.nodeId !== highlightedSuggestion.nodeName && (
                              <div className="flex justify-between">
                                <span className="text-gray-600 dark:text-gray-400">ID</span>
                                <span className="text-gray-700 dark:text-gray-300 truncate ml-2 max-w-[120px]">
                                  {highlightedSuggestion.nodeId}
                                </span>
                              </div>
                            )}
                        </div>
                      </>
                    ) : /* Function suggestions */
                    highlightedSuggestion?.kind === "function" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">
                          {highlightedSuggestion.label}()
                        </div>
                        {highlightedSuggestion.description && (
                          <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                            {highlightedSuggestion.description}
                          </div>
                        )}
                        {highlightedSuggestion.example && (
                          <div
                            className={twMerge(
                              valuePreviewCodeBlockClassName,
                              "text-green-700 dark:text-green-400 break-all",
                            )}
                          >
                            {highlightedSuggestion.example}
                          </div>
                        )}
                      </>
                    ) : /* Object values */
                    highlightedValue !== null &&
                      typeof highlightedValue === "object" &&
                      !Array.isArray(highlightedValue) ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Object</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                          {
                            Object.keys(highlightedValue as Record<string, unknown>).filter((k) => !k.startsWith("__"))
                              .length
                          }{" "}
                          properties
                        </div>
                        <div className={twMerge(valuePreviewCodeBlockClassName, "space-y-0.5")}>
                          {Object.keys(highlightedValue as Record<string, unknown>)
                            .filter((k) => !k.startsWith("__"))
                            .slice(0, 5)
                            .map((key) => (
                              <div key={key} className="truncate">
                                <span className="text-gray-500 dark:text-gray-400">.</span>
                                <span className="text-sky-700 dark:text-sky-400">{key}</span>
                              </div>
                            ))}
                          {Object.keys(highlightedValue as Record<string, unknown>).filter((k) => !k.startsWith("__"))
                            .length > 5 && (
                            <div className="text-gray-600 dark:text-gray-400 mt-1">
                              +
                              {Object.keys(highlightedValue as Record<string, unknown>).filter(
                                (k) => !k.startsWith("__"),
                              ).length - 5}{" "}
                              more...
                            </div>
                          )}
                        </div>
                      </>
                    ) : /* Array values */
                    Array.isArray(highlightedValue) ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Array</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                          {highlightedValue.length} item{highlightedValue.length !== 1 ? "s" : ""}
                        </div>
                        {highlightedValue.length > 0 && (
                          <div className={valuePreviewCodeBlockClassName}>
                            <span className="text-gray-500 dark:text-gray-400">[</span>
                            <span className="text-purple-700 dark:text-purple-400">{typeof highlightedValue[0]}</span>
                            <span className="text-gray-500 dark:text-gray-400">, ...]</span>
                          </div>
                        )}
                      </>
                    ) : /* String values */
                    typeof highlightedValue === "string" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">String</div>
                        {highlightedValue.length > 50 && (
                          <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                            {highlightedValue.length} characters
                          </div>
                        )}
                        <div className={twMerge(valuePreviewCodeBlockClassName, "break-all")}>
                          <span className="text-amber-700 dark:text-amber-400">"</span>
                          <span className="text-amber-800 dark:text-amber-300">
                            {highlightedValue.length > 100 ? highlightedValue.slice(0, 100) + "..." : highlightedValue}
                          </span>
                          <span className="text-amber-700 dark:text-amber-400">"</span>
                        </div>
                      </>
                    ) : /* Number values */
                    typeof highlightedValue === "number" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Number</div>
                        <div
                          className={twMerge(valuePreviewCodeBlockClassName, "text-orange-700 dark:text-orange-400")}
                        >
                          {highlightedValue}
                        </div>
                      </>
                    ) : /* Boolean values */
                    typeof highlightedValue === "boolean" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Boolean</div>
                        <div className={valuePreviewCodeBlockClassName}>
                          <span
                            className={
                              highlightedValue ? "text-green-700 dark:text-green-400" : "text-red-600 dark:text-red-400"
                            }
                          >
                            {String(highlightedValue)}
                          </span>
                        </div>
                      </>
                    ) : /* Null values */
                    highlightedValue === null ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Null</div>
                        <div
                          className={twMerge(valuePreviewCodeBlockClassName, "text-gray-600 dark:text-gray-400 italic")}
                        >
                          null
                        </div>
                      </>
                    ) : (
                      /* Fallback: show type */
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Type</div>
                        <div className={twMerge(valuePreviewCodeBlockClassName, "text-gray-700 dark:text-gray-300")}>
                          {highlightedSuggestion?.detail ?? highlightedSuggestion?.kind ?? "unknown"}
                        </div>
                      </>
                    )}
                  </div>
                )}
                <div
                  ref={suggestionsListRef}
                  className="overflow-auto bg-white border border-gray-200 dark:bg-gray-800 dark:border-gray-600 sm:rounded-r-lg rounded-b-lg sm:rounded-tl-none shadow-lg"
                  style={{
                    width: `${dropdownWidth}px`,
                    height: `${SUGGESTION_LIST_MAX_HEIGHT_PX}px`,
                  }}
                >
                  {(() => {
                    const nodeDataSuggestions = suggestions.filter(
                      (s) =>
                        ["$", "root", "previous"].includes(s.label) ||
                        s.nodeName ||
                        (s.kind !== "function" && s.kind !== "keyword"),
                    );
                    const functionSuggestions = suggestions.filter(
                      (s) => s.kind === "function" && !["root", "previous"].includes(s.label),
                    );

                    const renderSuggestionItem = (suggestionItem: Suggestion, index: number) => (
                      <div
                        key={`${suggestionItem.kind}-${suggestionItem.label}-${index}`}
                        data-suggestion-index={index}
                        className={twMerge([
                          "flex h-9 shrink-0 cursor-pointer items-center gap-2 px-3 text-sm leading-none",
                          "text-gray-950 dark:text-white",
                          highlightedIndex === index && "bg-slate-100 dark:bg-gray-700",
                        ])}
                        onMouseDown={(e) => {
                          isInteractingWithSuggestionsRef.current = true;
                          e.preventDefault(); // Prevent blur on the input
                          e.stopPropagation();
                          handleSuggestionClick(suggestionItem);
                        }}
                        onMouseEnter={(e) => {
                          e.stopPropagation();
                          highlightSuggestionAtIndex(index);
                        }}
                      >
                        <span className="truncate min-w-0">{getSuggestionDisplayLabel(suggestionItem)}</span>
                        {suggestionItem.kind === "function" && !["root", "previous"].includes(suggestionItem.label) && (
                          <span className="text-gray-500 dark:text-gray-400 truncate min-w-0">
                            {formatFunctionSignature(suggestionItem)}
                          </span>
                        )}
                        {["$", "root", "previous"].includes(suggestionItem.label) && (
                          <span className="inline-flex h-5 shrink-0 items-center rounded bg-blue-100 px-1.5 text-xs font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-300">
                            event data
                          </span>
                        )}
                        {suggestionItem.kind !== "function" && suggestionItem.labelDetail && (
                          <span className="inline-flex h-5 shrink-0 items-center rounded bg-blue-100 px-1.5 text-xs font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-300">
                            node
                          </span>
                        )}
                        <span className="inline-flex h-5 shrink-0 items-center rounded bg-slate-100 px-1.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                          {suggestionItem.detail ?? suggestionItem.kind}
                        </span>
                        <span className="ml-auto inline-flex h-5 shrink-0 items-center rounded border border-gray-300 px-1 text-[10px] leading-none text-gray-400 dark:border-gray-600 dark:text-gray-500">
                          Tab
                        </span>
                      </div>
                    );

                    // Calculate the starting index for functions (after node data suggestions)
                    const functionStartIndex = nodeDataSuggestions.length;

                    return (
                      <>
                        {nodeDataSuggestions.length > 0 && (
                          <>
                            <div
                              data-suggestion-section-header
                              className="sticky top-0 z-10 flex h-7 shrink-0 items-center border-b border-gray-200 bg-white dark:bg-gray-800 px-3 text-xs font-medium leading-none text-gray-500 dark:border-gray-600 dark:text-gray-400"
                            >
                              Connected nodes data
                            </div>
                            {nodeDataSuggestions.map((suggestionItem, idx) =>
                              renderSuggestionItem(suggestionItem, idx),
                            )}
                          </>
                        )}
                        {functionSuggestions.length > 0 && (
                          <>
                            <div
                              data-suggestion-section-header
                              className="sticky top-0 z-10 flex h-7 shrink-0 items-center border-b border-gray-200 bg-white dark:bg-gray-800 px-3 text-xs font-medium leading-none text-gray-500 dark:border-gray-600 dark:text-gray-400"
                            >
                              Expr functions
                            </div>
                            {functionSuggestions.map((suggestionItem, idx) =>
                              renderSuggestionItem(suggestionItem, functionStartIndex + idx),
                            )}
                          </>
                        )}
                      </>
                    );
                  })()}
                </div>
              </div>
            </div>,
            document.body,
          )}

        {/* Empty State */}
        {isOpen && suggestions.length === 0 && inputValue && (
          <div
            className={twMerge([
              "absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg",
              "dark:bg-gray-800 dark:border-gray-600",
            ])}
          >
            <div className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
              {!exampleObj ? noExampleObjectText : "No suggestions found"}
            </div>
          </div>
        )}
      </div>
    );
  },
);
