import React, { useState, useEffect, useRef, forwardRef, useImperativeHandle, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { twMerge } from "tailwind-merge";
import { getSuggestions, Suggestion } from "./core";
import { Eye, EyeOff } from "lucide-react";
import { evaluateExpr, formatExprResult } from "@/lib/exprEvaluator";

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
  quickTip?: string;
  expressionMode?: "wrapped" | "raw";
}

const suggestionSortPriority = {
  $: 1,
  root: 2,
  previous: 3,
} as const;

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
      quickTip,
      expressionMode = "wrapped",
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

    const isRawExpression = expressionMode === "raw";

    // Check if input contains any expressions
    const hasExpressions = useMemo(() => {
      if (isRawExpression) {
        return inputValue.trim().length > 0;
      }
      return /\{\{.*?\}\}/.test(inputValue);
    }, [inputValue, isRawExpression]);

    // Check if all expressions are valid
    const allExpressionsValid = useMemo(() => {
      if (!exampleObj) return true;
      if (isRawExpression) {
        if (inputValue.trim().length === 0) return true;
        try {
          evaluateExpr(inputValue.trim(), exampleObj);
          return true;
        } catch {
          return false;
        }
      }
      const regex = /\{\{(.*?)\}\}/g;
      let match;
      while ((match = regex.exec(inputValue)) !== null) {
        try {
          evaluateExpr(match[1].trim(), exampleObj);
        } catch {
          return false;
        }
      }
      return true;
    }, [inputValue, exampleObj, isRawExpression]);

    const containerRef = useRef<HTMLDivElement>(null);
    const suggestionsRef = useRef<HTMLDivElement>(null);
    const suggestionsListRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLTextAreaElement>(null);
    const backdropRef = useRef<HTMLDivElement>(null);
    const mirrorRef = useRef<HTMLSpanElement>(null);
    const isInteractingWithSuggestionsRef = useRef(false);
    const suppressSuggestionsRef = useRef(false);
    useImperativeHandle(forwardedRef, () => inputRef.current as HTMLTextAreaElement);

    // Auto-resize textarea based on content (and backdrop in preview mode)
    const adjustTextareaHeight = useCallback(() => {
      const textarea = inputRef.current;
      const backdrop = backdropRef.current;
      if (!textarea) return;
      textarea.style.height = "auto";
      // In preview mode, backdrop content may be longer than textarea content
      // Use the larger of the two heights
      const textareaHeight = textarea.scrollHeight;
      const backdropHeight = backdrop?.scrollHeight ?? 0;
      const finalHeight = Math.max(textareaHeight, backdropHeight);
      textarea.style.height = `${finalHeight}px`;
    }, []);

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
            <span key={key++} className="text-gray-500">
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

    // Evaluate an expression against the exampleObj using the expr evaluator
    const evaluateExpression = useCallback(
      (expr: string): { value: string; error?: string } => {
        if (!exampleObj) return { value: "?", error: "No context available" };
        try {
          const result = evaluateExpr(expr.trim(), exampleObj);
          return { value: formatExprResult(result) };
        } catch (e) {
          const errorMsg = e instanceof Error ? e.message : "Evaluation failed";
          return { value: "error", error: errorMsg };
        }
      },
      [exampleObj],
    );

    // Render content with highlighted expressions
    const renderHighlightedContent = (text: string) => {
      if (isRawExpression) {
        if (!text) {
          return [<span key={0}>{"\u200B"}</span>];
        }
        if (previewMode) {
          const result = evaluateExpression(text);
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
            <span key={key++} className="bg-gray-100 dark:bg-gray-800 rounded-sm">
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

    // Sync scroll between textarea and backdrop
    const handleScroll = useCallback(() => {
      if (inputRef.current && backdropRef.current) {
        backdropRef.current.scrollTop = inputRef.current.scrollTop;
        backdropRef.current.scrollLeft = inputRef.current.scrollLeft;
      }
    }, []);

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

    const isAllowedToSuggest = (text: string, position: number) => {
      if (isRawExpression || !props.startWord || !props.suffix) {
        return true;
      }

      const openIndex = text.lastIndexOf(props.startWord, position);
      if (openIndex === -1) {
        return false;
      }

      const closeIndex = text.indexOf(props.suffix, openIndex + 2);
      if (closeIndex !== -1 && position - 1 > closeIndex) {
        return false;
      }

      return true;
    };

    const getExpressionContext = (text: string, cursor: number) => {
      if (isRawExpression || !startWord || !suffix) {
        return {
          expressionText: text,
          expressionCursor: cursor,
          startOffset: 0,
          endOffset: text.length,
        };
      }

      const openIndex = text.lastIndexOf(startWord, cursor);
      if (openIndex === -1) {
        return null;
      }

      const closeIndex = text.indexOf(suffix, openIndex + startWord.length);
      if (closeIndex !== -1 && cursor - 1 > closeIndex) {
        return null;
      }

      const startOffset = openIndex + startWord.length;
      const endOffset = closeIndex === -1 ? text.length : closeIndex;
      return {
        expressionText: text.slice(startOffset, endOffset),
        expressionCursor: Math.max(0, cursor - startOffset),
        startOffset,
        endOffset,
      };
    };

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

    const resolveExpressionValue = (expression: string, globals: Record<string, unknown>) => {
      const tailExpr = extractTailExpressionWithParens(expression);
      if (!tailExpr) return undefined;

      const stripWhitespaceOutsideStrings = (input: string) => {
        let out = "";
        let inSingle = false;
        let inDouble = false;

        const isEscaped = (idx: number) => {
          let backslashes = 0;
          for (let j = idx - 1; j >= 0 && input[j] === "\\"; j--) {
            backslashes++;
          }
          return backslashes % 2 === 1;
        };

        for (let i = 0; i < input.length; i++) {
          const ch = input[i];
          if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
          else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

          if (!inSingle && !inDouble && /\s/u.test(ch)) {
            continue;
          }
          out += ch;
        }

        return out;
      };

      let expr = stripWhitespaceOutsideStrings(tailExpr);
      const normalized = normalizeSpecialFunctionExpr(expr);
      if (normalized === null) {
        return undefined;
      }
      expr = normalized;
      if (expr.startsWith("$[")) {
        expr = "$" + expr.slice(1);
      }

      type Token = { t: "dot" } | { t: "ident"; v: string } | { t: "key"; v: string };

      const tokens: Token[] = [];
      let i = 0;
      const identRe = /^[$A-Za-z_][$A-Za-z0-9_]*/;

      while (i < expr.length) {
        const rest = expr.slice(i);

        if (rest[0] === ".") {
          tokens.push({ t: "dot" });
          i += 1;
          continue;
        }

        if (rest[0] === "[") {
          const quotedMatch = rest.match(/^\[\s*(['"])(.*?)\1\s*\]/);
          if (quotedMatch) {
            tokens.push({ t: "key", v: String(quotedMatch[2] ?? "").replace(/\\(["'\\])/g, "$1") });
            i += quotedMatch[0].length;
            continue;
          }

          const numberMatch = rest.match(/^\[\s*(\d+)\s*\]/);
          if (numberMatch) {
            tokens.push({ t: "key", v: numberMatch[1] });
            i += numberMatch[0].length;
            continue;
          }

          return undefined;
          continue;
        }

        const im = rest.match(identRe);
        if (im) {
          tokens.push({ t: "ident", v: im[0] });
          i += im[0].length;
          continue;
        }

        return undefined;
      }

      if (tokens[0]?.t !== "ident") return undefined;
      let pos = 0;
      const first = (tokens[pos] as { t: "ident"; v: string }).v;
      pos += 1;

      let cur: unknown;
      if (first === "$" || first === "$env") cur = globals;
      else cur = globals ? (globals as Record<string, unknown>)[first] : undefined;

      while (pos < tokens.length) {
        const tok = tokens[pos];

        if (tok.t === "dot") {
          pos += 1;
          const next = tokens[pos];
          if (!next) return cur;
          if (next.t !== "ident") return undefined;
          try {
            cur = (cur as any)?.[next.v];
          } catch {
            return undefined;
          }
          pos += 1;
          continue;
        }

        if (tok.t === "key") {
          try {
            cur = (cur as any)?.[tok.v];
          } catch {
            return undefined;
          }
          pos += 1;
          continue;
        }

        return undefined;
      }

      return cur;
    };

    const extractTailExpressionWithParens = (expr: string) => {
      const s = expr.trim();
      let i = s.length - 1;
      let bracketDepth = 0;
      let parenDepth = 0;
      let inSingle = false;
      let inDouble = false;

      const isEscaped = (idx: number): boolean => {
        let bs = 0;
        for (let j = idx - 1; j >= 0 && s[j] === "\\"; j--) bs++;
        return bs % 2 === 1;
      };

      const isStopChar = (ch: string): boolean =>
        ch === "(" ||
        ch === ")" ||
        ch === "," ||
        ch === ";" ||
        ch === ":" ||
        ch === "?" ||
        ch === "+" ||
        ch === "-" ||
        ch === "*" ||
        ch === "/" ||
        ch === "%" ||
        ch === "|" ||
        ch === "&" ||
        ch === "!" ||
        ch === "=" ||
        ch === "<" ||
        ch === ">" ||
        ch === "\n" ||
        ch === "\r" ||
        ch === "\t" ||
        ch === " ";

      for (; i >= 0; i--) {
        const ch = s[i];

        if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
        else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

        if (inSingle || inDouble) continue;

        if (ch === "]") {
          bracketDepth++;
          continue;
        }
        if (ch === "[") {
          bracketDepth = Math.max(0, bracketDepth - 1);
          continue;
        }
        if (ch === ")") {
          parenDepth++;
          continue;
        }
        if (ch === "(") {
          if (parenDepth == 0) {
            return s.slice(i + 1).trim();
          }
          parenDepth = Math.max(0, parenDepth - 1);
          continue;
        }

        if (bracketDepth > 0 || parenDepth > 0) continue;

        if (isStopChar(ch)) {
          return s.slice(i + 1).trim();
        }
      }

      return s;
    };

    const normalizeSpecialFunctionExpr = (expr: string): string | null => {
      const rootMatch = expr.match(/^root\(\)/);
      if (rootMatch) {
        return `__root${expr.slice(rootMatch[0].length)}`;
      }

      const previousMatch = expr.match(/^previous\(([^)]*)\)/);
      if (previousMatch) {
        const raw = (previousMatch[1] ?? "").trim();
        const depth = raw === "" ? 1 : Number(raw);
        if (!Number.isInteger(depth) || depth < 1) {
          return null;
        }

        return `__previousByDepth["${depth}"]${expr.slice(previousMatch[0].length)}`;
      }

      return expr;
    };

    const computeHighlightedValue = React.useCallback(
      (suggestion: ReturnType<typeof getSuggestions>[number], context: ReturnType<typeof getExpressionContext>) => {
        if (!exampleObj || !context) return undefined;
        if (suggestion.kind === "function") return undefined;

        const insertText = getSuggestionInsertText(suggestion);
        const left = context.expressionText.slice(0, context.expressionCursor);
        const range = getReplacementRange(left, insertText);
        const nextExpressionLeft = context.expressionText.slice(0, range.start) + insertText;
        const value = resolveExpressionValue(nextExpressionLeft, exampleObj);
        if (typeof value === "function") return undefined;
        return value;
      },
      [exampleObj],
    );

    const valuePreviewWidth = 200;

    const renderQuickTip = (tip: string) => {
      const parts = tip.split(/`([^`]+)`/g);
      return parts.map((part, index) =>
        index % 2 === 1 ? (
          <code
            key={`code-${index}`}
            className="bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded text-gray-700 dark:text-gray-300"
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

      // Copy relevant styles to mirror element
      mirror.style.font = computed.font;
      mirror.style.fontSize = computed.fontSize;
      mirror.style.fontFamily = computed.fontFamily;
      mirror.style.fontWeight = computed.fontWeight;
      mirror.style.letterSpacing = computed.letterSpacing;
      mirror.style.textTransform = computed.textTransform;

      // Set content to text before cursor
      const textBeforeCursor = inputValue.substring(0, cursorPosition);
      mirror.textContent = textBeforeCursor || "\u200b"; // Use zero-width space if empty

      // Measure the width and account for input's left padding
      const paddingLeft = parseFloat(computed.paddingLeft) || 0;
      const cursorOffset = mirror.offsetWidth + paddingLeft;

      // Calculate cursor position relative to viewport
      const inputRect = input.getBoundingClientRect();
      const cursorScreenX = inputRect.left + cursorOffset;
      const viewportWidth = window.innerWidth;
      const edgePadding = 16; // Padding from screen edge

      // Space available on each side of cursor
      const spaceOnRight = viewportWidth - cursorScreenX - edgePadding;
      const spaceOnLeft = cursorScreenX - edgePadding;

      // Determine if we should flip based on available space
      // Normal: suggestions start at cursor (need dropdownWidth on right)
      // Flipped: suggestions end at cursor (need dropdownWidth on left)
      const shouldFlipLeft = spaceOnRight < dropdownWidth && spaceOnLeft >= dropdownWidth;

      // Calculate absolute position for portal
      const dropdownTop = inputRect.bottom + 4; // 4px gap below input
      let dropdownLeft: number;

      if (shouldFlipLeft) {
        // Flipped: suggestions end at cursor, Value Preview extends further left
        dropdownLeft = showValuePreview
          ? cursorScreenX - dropdownWidth - valuePreviewWidth
          : cursorScreenX - dropdownWidth;
      } else {
        // Normal: suggestions start at cursor, Value Preview is to the left of cursor
        dropdownLeft = showValuePreview ? cursorScreenX - valuePreviewWidth : cursorScreenX;
      }

      // Clamp to screen edges to prevent overflow
      const totalWidth = showValuePreview ? dropdownWidth + valuePreviewWidth : dropdownWidth;
      dropdownLeft = Math.max(edgePadding, Math.min(dropdownLeft, viewportWidth - totalWidth - edgePadding));

      setDropdownPosition({
        top: dropdownTop,
        left: dropdownLeft,
      });
    }, [inputValue, cursorPosition, dropdownWidth, showValuePreview]);

    // Measure cursor pixel position when cursor or input changes
    useEffect(() => {
      measureCursorPixelPosition();
    }, [measureCursorPixelPosition]);

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
      if (!isFocused) {
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      if (suppressSuggestionsRef.current) {
        suppressSuggestionsRef.current = false;
        return;
      }

      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context || !isAllowedToSuggest(inputValue, cursorPosition)) {
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      const newSuggestions = getSuggestions(context.expressionText, context.expressionCursor, exampleObj ?? {}, {
        limit: 150,
      }).sort((a, b) => {
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
      });
      setSuggestions(newSuggestions);
      setIsOpen(newSuggestions.length > 0);
      const nextHighlightedIndex = showValuePreview && newSuggestions.length > 0 ? 0 : -1;
      setHighlightedIndex(nextHighlightedIndex);
      if (nextHighlightedIndex >= 0) {
        const suggestion = newSuggestions[nextHighlightedIndex];
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
      computeHighlightedValue,
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

    const handleSuggestionClick = (suggestionItem: ReturnType<typeof getSuggestions>[number]) => {
      suppressSuggestionsRef.current = true;
      const cursorPosition = inputRef.current?.selectionStart || 0;
      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context) {
        suppressSuggestionsRef.current = false;
        isInteractingWithSuggestionsRef.current = false;
        setIsOpen(false);
        return;
      }

      const left = context.expressionText.slice(0, context.expressionCursor);
      const insertText = getSuggestionInsertText(suggestionItem);
      const range = getReplacementRange(left, insertText);
      const nextExpression =
        context.expressionText.slice(0, range.start) + insertText + context.expressionText.slice(range.end);
      const newValue = inputValue.slice(0, context.startOffset) + nextExpression + inputValue.slice(context.endOffset);

      setInputValue(newValue);
      onChange?.(newValue);
      setHighlightedIndex(-1);
      requestAnimationFrame(() => {
        inputRef.current?.focus();
        const cursorTarget = context.startOffset + range.start + insertText.length;
        inputRef.current?.setSelectionRange(cursorTarget, cursorTarget);
        isInteractingWithSuggestionsRef.current = false;
        suppressSuggestionsRef.current = false;
      });

      setIsOpen(false);
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!isOpen || suggestions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setHighlightedIndex((prev) => {
            const newIndex = prev < suggestions.length - 1 ? prev + 1 : 0;
            if (suggestions[newIndex]) {
              setHighlightedSuggestion(suggestions[newIndex]);
              const cursorPosition = inputRef.current?.selectionStart || 0;
              const context = getExpressionContext(inputValue, cursorPosition);
              const value = computeHighlightedValue(suggestions[newIndex], context);
              setHighlightedValue(value);
            }
            return newIndex;
          });
          break;
        case "ArrowUp":
          e.preventDefault();
          setHighlightedIndex((prev) => {
            const newIndex = prev > 0 ? prev - 1 : suggestions.length - 1;
            if (suggestions[newIndex]) {
              setHighlightedSuggestion(suggestions[newIndex]);
              const cursorPosition = inputRef.current?.selectionStart || 0;
              const context = getExpressionContext(inputValue, cursorPosition);
              const value = computeHighlightedValue(suggestions[newIndex], context);
              setHighlightedValue(value);
            }
            return newIndex;
          });
          break;
        case "Enter":
          if (highlightedIndex >= 0) {
            e.preventDefault();
            handleSuggestionClick(suggestions[highlightedIndex]);
          }
          // Allow default behavior (newline) when no suggestion is highlighted
          break;
        case "Tab":
          if (highlightedIndex >= 0) {
            e.preventDefault();
            handleSuggestionClick(suggestions[highlightedIndex]);
          }
          break;
        case "Escape":
          setIsOpen(false);
          setHighlightedIndex(-1);
          setHighlightedValue(undefined);
          setHighlightedSuggestion(null);
          break;
      }
    };

    // Scroll highlighted item into view
    useEffect(() => {
      if (highlightedIndex >= 0 && suggestionsListRef.current) {
        const highlightedElement = suggestionsListRef.current.querySelector(
          `[data-suggestion-index="${highlightedIndex}"]`,
        ) as HTMLElement;
        if (highlightedElement) {
          highlightedElement.scrollIntoView({
            block: "nearest",
          });
        }
      }
    }, [highlightedIndex]);

    // Always show Value Preview box when enabled and something is highlighted
    // to prevent position jumping when switching between suggestion types
    const shouldShowValuePreview = showValuePreview && highlightedIndex >= 0;

    return (
      <div ref={containerRef} className={"relative w-full" + (quickTip || hasExpressions ? " mb-6" : "")}>
        {/* Hidden mirror element for measuring cursor position */}
        <span
          ref={mirrorRef}
          aria-hidden="true"
          style={{
            position: "absolute",
            visibility: "hidden",
            whiteSpace: "pre",
            pointerEvents: "none",
            top: 0,
            left: 0,
          }}
        />
        {/* Input Field with Syntax Highlighting */}
        <span
          data-slot="control"
          className={twMerge([
            "relative block w-full",
            "focus-within:ring-ring/50",
            "has-data-disabled:opacity-50",
            className,
          ])}
        >
          {/* Backdrop for syntax highlighting */}
          <div
            ref={backdropRef}
            aria-hidden="true"
            className={twMerge([
              "font-sm pointer-events-none absolute inset-0 whitespace-pre-wrap break-words overflow-hidden",
              "rounded-md border border-transparent px-3 py-2 text-base",
              "text-gray-950 dark:text-white",
              // Size variants (must match textarea)
              inputSize === "xs" && "min-h-7 px-2 text-xs",
              inputSize === "sm" && "min-h-8 px-2 text-sm",
              inputSize === "md" && "min-h-8 px-3 text-base md:text-sm",
              inputSize === "lg" && "min-h-11 px-4 text-lg",
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
            onKeyDownCapture={handleCursorChange}
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
              "font-sm bg-transparent border-gray-300 shadow-xs placeholder:text-gray-500 selection:bg-primary/30",
              "relative block w-full min-w-0 appearance-none rounded-md border px-3 py-2 text-base outline-none resize-none overflow-hidden",
              "focus-visible:border-gray-500 focus-visible:ring-ring/50",
              "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
              "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
              // Make text transparent but keep caret visible
              "text-transparent caret-gray-950 dark:caret-white",
              // Size variants (min-height instead of fixed height)
              inputSize === "xs" && "min-h-7 px-2 text-xs",
              inputSize === "sm" && "min-h-8 px-2 text-sm",
              inputSize === "md" && "min-h-8 px-3 text-base md:text-sm",
              inputSize === "lg" && "min-h-11 px-4 text-lg",
            ])}
            {...rest}
          />
          {/* Bottom bar with preview toggle and quickTip */}
          {(hasExpressions || quickTip) && (
            <div className="absolute -bottom-6 left-0 right-0 flex items-center justify-between">
              {/* Preview toggle - left side */}
              {hasExpressions ? (
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
                  <span>Preview</span>
                </button>
              ) : (
                <span />
              )}
              {/* QuickTip - right side */}
              {(quickTip || hasExpressions) && (
                <span className="flex items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
                  {quickTip
                    ? renderQuickTip(quickTip)
                    : [
                        "Use ",
                        <code
                          key="default-tip"
                          className="bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded text-gray-700 dark:text-gray-300"
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
              )}
            </div>
          )}
        </span>

        {/* Suggestions Dropdown - rendered in portal to escape overflow:hidden */}
        {isOpen &&
          suggestions.length > 0 &&
          createPortal(
            <div
              ref={suggestionsRef}
              className="fixed z-[9999] bg-transparent"
              style={{
                top: `${dropdownPosition.top}px`,
                left: `${dropdownPosition.left}px`,
              }}
            >
              <div className="flex flex-col sm:flex-row">
                {shouldShowValuePreview && isOpen && (
                  <div
                    className="border border-gray-200 dark:border-gray-700 sm:border-r-0 sm:border-t p-3 bg-gray-100 dark:bg-gray-700 sm:rounded-l-lg rounded-t-lg sm:rounded-br-none h-fit self-start shadow-lg"
                    style={{ width: `${valuePreviewWidth}px` }}
                  >
                    {/* $ selector */}
                    {highlightedSuggestion?.label === "$" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">$ (Event Data)</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                          Root selector for accessing payload data from all connected components.
                        </div>
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 text-sky-400">
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
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 space-y-1">
                          <div className="flex justify-between">
                            <span className="text-gray-500">Name</span>
                            <span className="text-sky-400 truncate ml-2 max-w-[120px]">
                              {highlightedSuggestion.nodeName}
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-gray-500">ID</span>
                            <span className="text-gray-400 truncate ml-2 max-w-[120px]">
                              {highlightedSuggestion.nodeId}
                            </span>
                          </div>
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
                          <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 text-emerald-400 break-all">
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
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 space-y-0.5">
                          {Object.keys(highlightedValue as Record<string, unknown>)
                            .filter((k) => !k.startsWith("__"))
                            .slice(0, 5)
                            .map((key) => (
                              <div key={key} className="truncate">
                                <span className="text-gray-400">.</span>
                                <span className="text-sky-400">{key}</span>
                              </div>
                            ))}
                          {Object.keys(highlightedValue as Record<string, unknown>).filter((k) => !k.startsWith("__"))
                            .length > 5 && (
                            <div className="text-gray-500 mt-1">
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
                          <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2">
                            <span className="text-gray-400">[</span>
                            <span className="text-purple-400">{typeof highlightedValue[0]}</span>
                            <span className="text-gray-400">, ...]</span>
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
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 break-all">
                          <span className="text-amber-400">"</span>
                          <span className="text-amber-300">
                            {highlightedValue.length > 100 ? highlightedValue.slice(0, 100) + "..." : highlightedValue}
                          </span>
                          <span className="text-amber-400">"</span>
                        </div>
                      </>
                    ) : /* Number values */
                    typeof highlightedValue === "number" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Number</div>
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 text-orange-400">
                          {highlightedValue}
                        </div>
                      </>
                    ) : /* Boolean values */
                    typeof highlightedValue === "boolean" ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Boolean</div>
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2">
                          <span className={highlightedValue ? "text-green-400" : "text-red-400"}>
                            {String(highlightedValue)}
                          </span>
                        </div>
                      </>
                    ) : /* Null values */
                    highlightedValue === null ? (
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Null</div>
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 text-gray-500 italic">
                          null
                        </div>
                      </>
                    ) : (
                      /* Fallback: show type */
                      <>
                        <div className="text-sm font-medium text-gray-950 dark:text-white mb-1">Type</div>
                        <div className="text-xs font-mono bg-gray-900 dark:bg-gray-900 rounded px-2.5 py-2 text-gray-300">
                          {highlightedSuggestion?.detail ?? highlightedSuggestion?.kind ?? "unknown"}
                        </div>
                      </>
                    )}
                  </div>
                )}
                <div
                  ref={suggestionsListRef}
                  className="overflow-auto bg-white border border-gray-200 dark:bg-gray-800 dark:border-gray-700 sm:rounded-r-lg rounded-b-lg sm:rounded-tl-none max-h-60 shadow-lg"
                  style={{ width: `${dropdownWidth}px` }}
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
                          "px-3 py-2 cursor-pointer text-sm flex items-center gap-2",
                          "hover:bg-gray-100 dark:hover:bg-gray-700",
                          "text-gray-950 dark:text-white",
                          highlightedIndex === index && "bg-gray-100 dark:bg-gray-700",
                        ])}
                        onMouseDown={(e) => {
                          isInteractingWithSuggestionsRef.current = true;
                          e.preventDefault(); // Prevent blur on the input
                          e.stopPropagation();
                          handleSuggestionClick(suggestionItem);
                        }}
                        onMouseEnter={(e) => {
                          e.stopPropagation();
                          setHighlightedIndex(index);
                          setHighlightedSuggestion(suggestionItem);
                          if (exampleObj) {
                            const cursorPosition = inputRef.current?.selectionStart || 0;
                            const context = getExpressionContext(inputValue, cursorPosition);
                            const value = computeHighlightedValue(suggestionItem, context);
                            setHighlightedValue(value);
                          }
                        }}
                      >
                        <span className="truncate min-w-0">{suggestionItem.label}</span>
                        {suggestionItem.kind === "function" && (
                          <span className="text-gray-500 truncate min-w-0">
                            {formatFunctionSignature(suggestionItem)}
                          </span>
                        )}
                        {["$", "root", "previous"].includes(suggestionItem.label) && (
                          <span className="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded">
                            event data
                          </span>
                        )}
                        {suggestionItem.kind !== "function" && suggestionItem.labelDetail && (
                          <span className="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded">
                            node
                          </span>
                        )}
                        <span className="flex-shrink-0 px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300 rounded">
                          {suggestionItem.detail ?? suggestionItem.kind}
                        </span>
                        <span className="ml-auto flex-shrink-0 text-[10px] text-gray-400 dark:text-gray-500 border border-gray-300 dark:border-gray-600 rounded px-1 py-0.5">
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
                            <div className="px-3 py-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 sticky top-0">
                              Connected nodes data
                            </div>
                            {nodeDataSuggestions.map((suggestionItem, idx) =>
                              renderSuggestionItem(suggestionItem, idx),
                            )}
                          </>
                        )}
                        {functionSuggestions.length > 0 && (
                          <>
                            <div className="px-3 py-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 sticky top-0">
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
              "dark:bg-gray-800 dark:border-gray-700",
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
