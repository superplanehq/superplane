import React, { useState, useEffect, useRef, forwardRef, useImperativeHandle } from "react";
import { twMerge } from "tailwind-merge";
import { getSuggestions } from "./core";

export interface AutoCompleteInputProps extends Omit<React.ComponentPropsWithoutRef<"input">, "onChange" | "size"> {
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
}

export const AutoCompleteInput = forwardRef<HTMLInputElement, AutoCompleteInputProps>(
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
      ...rest
    } = props;
    const [inputValue, setInputValue] = useState(value);
    const [suggestions, setSuggestions] = useState<Array<ReturnType<typeof getSuggestions>[number]>>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    const [highlightedIndex, setHighlightedIndex] = useState(-1);
    const [highlightedValue, setHighlightedValue] = useState<unknown>(undefined);
    const [cursorPosition, setCursorPosition] = useState(0);
    const previousWordLength = useRef<number>(0);
    const previousInputValue = useRef<string>(value);

    const containerRef = useRef<HTMLDivElement>(null);
    const suggestionsRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    useImperativeHandle(forwardedRef, () => inputRef.current as HTMLInputElement);

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
      if (!props.startWord || !props.suffix) {
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
      if (!startWord || !suffix) {
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
      if (closeIndex !== -1 && cursor -1 > closeIndex) {
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

    const extractTailPathExpression = (expr: string) => {
      const s = expr.trim();
      let bracketDepth = 0;
      let inSingle = false;
      let inDouble = false;

      const isEscaped = (idx: number) => {
        let backslashes = 0;
        for (let j = idx - 1; j >= 0 && s[j] === "\\"; j--) {
          backslashes++;
        }
        return backslashes % 2 === 1;
      };

      const isStopChar = (ch: string) =>
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

      let start = -1;
      for (let i = 0; i < s.length; i++) {
        const ch = s[i];

        if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
        else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

        if (inSingle || inDouble) continue;

        if (ch === "[") {
          bracketDepth++;
          continue;
        }
        if (ch === "]") {
          bracketDepth = Math.max(0, bracketDepth - 1);
          continue;
        }

        if (bracketDepth === 0 && ch === "$") {
          start = i;
        }
      }

      if (start === -1) return "";

      bracketDepth = 0;
      inSingle = false;
      inDouble = false;
      let end = s.length;

      for (let i = start; i < s.length; i++) {
        const ch = s[i];

        if (!inDouble && ch === "'" && !isEscaped(i)) inSingle = !inSingle;
        else if (!inSingle && ch === '"' && !isEscaped(i)) inDouble = !inDouble;

        if (inSingle || inDouble) continue;

        if (ch === "[") {
          bracketDepth++;
          continue;
        }
        if (ch === "]") {
          bracketDepth = Math.max(0, bracketDepth - 1);
          continue;
        }

        if (bracketDepth === 0 && i > start && isStopChar(ch)) {
          end = i;
          break;
        }
      }

      return s.slice(start, end).trim();
    };

    const resolveExpressionValue = (expression: string, globals: Record<string, unknown>) => {
      const tailExpr = extractTailPathExpression(expression);
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

    useEffect(() => {
      setInputValue(value);
    }, [value]);

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

      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context || !isAllowedToSuggest(inputValue, cursorPosition)) {
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      const newSuggestions = getSuggestions(context.expressionText, context.expressionCursor, exampleObj ?? {});
      setSuggestions(newSuggestions);
      setIsOpen(newSuggestions.length > 0);
      const nextHighlightedIndex = showValuePreview && newSuggestions.length > 0 ? 0 : -1;
      setHighlightedIndex(nextHighlightedIndex);
      if (exampleObj && nextHighlightedIndex >= 0) {
        const suggestion = newSuggestions[nextHighlightedIndex];
        const insertText = getSuggestionInsertText(suggestion);
        const left = context.expressionText.slice(0, context.expressionCursor);
        const range = getReplacementRange(left, insertText);
        const nextExpression =
          context.expressionText.slice(0, range.start) + insertText + context.expressionText.slice(range.end);
        const value = resolveExpressionValue(nextExpression, exampleObj);
        setHighlightedValue(value);
      } else {
        setHighlightedValue(undefined);
      }
    }, [inputValue, cursorPosition, isFocused, startWord, suffix, onChange, showValuePreview, exampleObj]);

    // Handle clicking outside to close suggestions
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
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

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
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
      const cursorPosition = inputRef.current?.selectionStart || 0;
      const context = getExpressionContext(inputValue, cursorPosition);
      if (!context) {
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
        const cursorTarget = context.startOffset + range.start + insertText.length;
        inputRef.current?.setSelectionRange(cursorTarget, cursorTarget);
      });

      setIsOpen(false);
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!isOpen || suggestions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setHighlightedIndex((prev) => {
            const newIndex = prev < suggestions.length - 1 ? prev + 1 : 0;
            if (exampleObj && suggestions[newIndex]) {
              const cursorPosition = inputRef.current?.selectionStart || 0;
              const context = getExpressionContext(inputValue, cursorPosition);
              if (!context) {
                setHighlightedValue(undefined);
                return newIndex;
              }
              const insertText = getSuggestionInsertText(suggestions[newIndex]);
              const left = context.expressionText.slice(0, context.expressionCursor);
              const range = getReplacementRange(left, insertText);
              const nextExpression =
                context.expressionText.slice(0, range.start) + insertText + context.expressionText.slice(range.end);
              const value = resolveExpressionValue(nextExpression, exampleObj);
              setHighlightedValue(value);
            }
            return newIndex;
          });
          break;
        case "ArrowUp":
          e.preventDefault();
          setHighlightedIndex((prev) => {
            const newIndex = prev > 0 ? prev - 1 : suggestions.length - 1;
            if (exampleObj && suggestions[newIndex]) {
              const cursorPosition = inputRef.current?.selectionStart || 0;
              const context = getExpressionContext(inputValue, cursorPosition);
              if (!context) {
                setHighlightedValue(undefined);
                return newIndex;
              }
              const insertText = getSuggestionInsertText(suggestions[newIndex]);
              const left = context.expressionText.slice(0, context.expressionCursor);
              const range = getReplacementRange(left, insertText);
              const nextExpression =
                context.expressionText.slice(0, range.start) + insertText + context.expressionText.slice(range.end);
              const value = resolveExpressionValue(nextExpression, exampleObj);
              setHighlightedValue(value);
            }
            return newIndex;
          });
          break;
        case "Enter":
          e.preventDefault();
          if (highlightedIndex >= 0) {
            handleSuggestionClick(suggestions[highlightedIndex]);
          }
          break;
        case "Escape":
          setIsOpen(false);
          setHighlightedIndex(-1);
          setHighlightedValue(undefined);
          break;
      }
    };

    // Scroll highlighted item into view
    useEffect(() => {
      if (highlightedIndex >= 0 && suggestionsRef.current) {
        const highlightedElement = suggestionsRef.current.children[highlightedIndex] as HTMLElement;
        if (highlightedElement) {
          highlightedElement.scrollIntoView({
            block: "nearest",
          });
        }
      }
    }, [highlightedIndex]);

    return (
      <div ref={containerRef} className="relative w-full">
        {/* Input Field */}
        <span
          data-slot="control"
          className={twMerge([
            "relative block w-full",
            "focus-within:ring-ring/50",
            "has-data-disabled:opacity-50",
            className,
          ])}
        >
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onKeyUp={handleCursorChange}
            onKeyDownCapture={handleCursorChange}
            onKeyUp={handleCursorChange}
            onClick={handleCursorChange}
            onSelect={handleCursorChange}
            onFocus={() => {
              setIsFocused(true);
              if (suggestions.length > 0) {
                setIsOpen(true);
              }
            }}
            onBlur={() => {
              // Small delay to allow click on suggestions
              setTimeout(() => {
                setIsFocused(false);
                setIsOpen(false);
                setHighlightedValue(undefined);
              }, 150);
            }}
            placeholder={placeholder}
            disabled={disabled}
            className={twMerge([
              "font-sm bg-white border-gray-300 shadow-xs file:text-foreground placeholder:text-gray-500 selection:bg-primary selection:text-primary-foreground",
              "relative block w-full min-w-0 appearance-none rounded-md border px-3 py-1 text-base outline-none",
              "file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium",
              "focus-visible:border-gray-500 focus-visible:ring-ring/50",
              "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
              "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
              // Size variants
              inputSize === "xs" && "h-7 px-2 text-xs",
              inputSize === "sm" && "h-8 px-2 text-sm",
              inputSize === "md" && "h-9 px-3 text-base md:text-sm",
              inputSize === "lg" && "h-11 px-4 text-lg",
            ])}
            {...rest}
          />
          {quickTip && (
            <span className="pointer-events-none absolute -bottom-4 right-1 text-[10px] font-medium text-gray-400 bg-gray-100 rounded-b-md px-2">
              {quickTip}
            </span>
          )}
        </span>

        {/* Value Preview Box */}
        {showValuePreview &&
          highlightedIndex >= 0 &&
          highlightedValue !== undefined &&
          isOpen &&
          (highlightedValue === null || (typeof highlightedValue !== "object" && !Array.isArray(highlightedValue))) && (
            <div
              className={twMerge([
                "absolute z-50 w-full bottom-full mb-1 bg-white border border-gray-200 rounded-lg shadow-lg p-3",
                "dark:bg-gray-800 dark:border-gray-700",
              ])}
            >
              <div className="text-xs text-gray-500 dark:text-gray-300 mb-1">Value Preview:</div>
              <div className="text-sm text-gray-950 dark:text-white font-mono break-all">
                {highlightedValue === null
                  ? "null"
                  : typeof highlightedValue === "string"
                    ? `"${highlightedValue}"`
                    : String(highlightedValue)}
              </div>
            </div>
          )}

        {/* Suggestions Dropdown */}
        {isOpen && suggestions.length > 0 && (
          <div
            ref={suggestionsRef}
            className={twMerge([
              "absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-auto",
              "dark:bg-gray-800 dark:border-gray-700",
            ])}
          >
            {suggestions.map((suggestionItem, index) => (
              <div
                key={`${suggestionItem.kind}-${suggestionItem.label}-${index}`}
                className={twMerge([
                  "px-3 py-2 cursor-pointer text-sm flex justify-between items-center",
                  "hover:bg-gray-100 dark:hover:bg-gray-700",
                  "text-gray-950 dark:text-white",
                  highlightedIndex === index && "bg-gray-100 dark:bg-gray-700",
                ])}
                onClick={() => handleSuggestionClick(suggestionItem)}
                onMouseEnter={() => {
                  setHighlightedIndex(index);
                  if (exampleObj) {
                    const cursorPosition = inputRef.current?.selectionStart || 0;
                    const context = getExpressionContext(inputValue, cursorPosition);
                    if (!context) {
                      setHighlightedValue(undefined);
                      return;
                    }
                    const insertText = getSuggestionInsertText(suggestionItem);
                    const left = context.expressionText.slice(0, context.expressionCursor);
                    const range = getReplacementRange(left, insertText);
                    const nextExpression =
                      context.expressionText.slice(0, range.start) +
                      insertText +
                      context.expressionText.slice(range.end);
                    const value = resolveExpressionValue(nextExpression, exampleObj);
                    setHighlightedValue(value);
                  }
                }}
              >
                <span>
                  {suggestionItem.label}
                  {suggestionItem.kind === "function" && (
                    <span className="ml-2 text-gray-500">{formatFunctionSignature(suggestionItem)}</span>
                  )}
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                  {suggestionItem.detail ?? suggestionItem.kind}
                </span>
              </div>
            ))}
          </div>
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
