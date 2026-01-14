import React, { useState, useEffect, useRef, forwardRef, useImperativeHandle } from "react";
import { twMerge } from "tailwind-merge";
import {
  buildLookupPath,
  formatDisplayPath,
  flattenForAutocomplete,
  getAutocompleteSuggestions,
  getAutocompleteSuggestionsWithTypes,
  getValueAtPath,
  isValidIdentifier,
  parsePathSegments,
} from "./core";

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
}

let blurTimeout: NodeJS.Timeout;

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
      ...rest
    } = props;
    const [inputValue, setInputValue] = useState(value);
    const [suggestions, setSuggestions] = useState<Array<{ suggestion: string; type: string }>>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    const [highlightedIndex, setHighlightedIndex] = useState(-1);
    const [flattenedData, setFlattenedData] = useState<Record<string, string[]>>({});
    const [highlightedValue, setHighlightedValue] = useState<unknown>(undefined);
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

    const isInsideDoubleBraces = (text: string, position: number) => {
      const openIndex = text.lastIndexOf("{{", position);
      if (openIndex === -1) {
        return false;
      }

      const closeIndex = text.indexOf("}}", openIndex + 2);
      if (closeIndex !== -1 && position > closeIndex) {
        return false;
      }

      return true;
    };

    const getNormalizedPath = (rawWord: string) => {
      return buildLookupPath(parsePathSegments(rawWord));
    };

    const getPathContext = (rawWord: string, normalizedWord: string) => {
      if (!normalizedWord) {
        return { basePath: "", lastKey: "" };
      }

      if (rawWord.endsWith(".")) {
        return { basePath: normalizedWord, lastKey: "" };
      }

      const parts = normalizedWord.split(".");
      return {
        basePath: parts.slice(0, -1).join("."),
        lastKey: parts[parts.length - 1] ?? "",
      };
    };

    // Helper function to replace word at cursor position
    const replaceWordAtCursor = (text: string, position: number, newWord: string) => {
      const { start, end } = getWordAtCursor(text, position);
      return text.substring(0, start) + newWord + text.substring(end);
    };

    // Helper function to build full path for a suggestion
    const buildFullPath = (suggestion: string) => {
      const cursorPosition = inputRef.current?.selectionStart || 0;
      const { word } = getWordAtCursor(inputValue, cursorPosition);
      const normalizedWord = getNormalizedPath(word);
      const { basePath } = getPathContext(word, normalizedWord);

      return suggestion.startsWith(basePath) ? suggestion : basePath ? `${basePath}.${suggestion}` : suggestion;
    };

    const formatSuggestionLabel = (suggestion: string) => {
      if (suggestion.match(/\[/)) {
        return suggestion;
      }
      if (isValidIdentifier(suggestion)) {
        return suggestion;
      }
      return `["${suggestion}"]`;
    };

    // Flatten the example object when it changes
    useEffect(() => {
      if (exampleObj) {
        const flattened = flattenForAutocomplete(exampleObj);
        setFlattenedData(flattened);
      }
    }, [exampleObj]);

    useEffect(() => {
      setInputValue(value);
    }, [value]);

    useEffect(() => {
      previousInputValue.current = inputValue;
    }, [inputValue]);

    useEffect(() => {
      if (!flattenedData || !isFocused) {
        setSuggestions([]);
        setIsOpen(false);
        return;
      }

      const cursorPosition = inputRef.current?.selectionStart || 0;
      const prevChar = cursorPosition > 0 ? inputValue[cursorPosition - 1] : "";
      const { word } = getWordAtCursor(inputValue, cursorPosition);

      if (word === "") {
        previousWordLength.current = 0;
        setSuggestions([]);
        setIsOpen(false);
        return;
      }

      if (prevChar === " ") {
        previousWordLength.current = word.length;
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      if (word.startsWith("[") && !word.startsWith("$")) {
        previousWordLength.current = word.length;
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      if (!isInsideDoubleBraces(inputValue, cursorPosition)) {
        previousWordLength.current = word.length;
        setSuggestions([]);
        setIsOpen(false);
        setHighlightedValue(undefined);
        return;
      }

      const normalizedWord = getNormalizedPath(word);
      const { basePath, lastKey } = getPathContext(word, normalizedWord);
      const parsedInput = basePath;

      const newSuggestions = getAutocompleteSuggestionsWithTypes(
        flattenedData,
        parsedInput || "root",
        basePath,
        exampleObj,
      );
      const arraySuggestions = getAutocompleteSuggestionsWithTypes(
        flattenedData,
        parsedInput ? `${parsedInput}.${lastKey}` : lastKey,
        basePath,
        exampleObj,
      ).filter(({ suggestion }) => suggestion.match(/\[\d+\]$/));
      const similarSuggestions = newSuggestions.filter(
        ({ suggestion }) => suggestion.startsWith(lastKey) && suggestion !== lastKey,
      );

      // Merge suggestions and remove duplicates based on suggestion text
      const allSuggestionsMap = new Map();
      [...arraySuggestions, ...similarSuggestions].forEach((item) => {
        allSuggestionsMap.set(item.suggestion, item);
      });
      const allSuggestions = Array.from(allSuggestionsMap.values());

      setSuggestions(allSuggestions);
      setIsOpen(
        allSuggestions.length > 0 ||
          (allSuggestions.length === 0 && word.endsWith(".")) ||
          word === "$" ||
          (!exampleObj && word.endsWith(".")),
      );
      const nextHighlightedIndex = showValuePreview && allSuggestions.length > 0 ? 0 : -1;
      setHighlightedIndex(nextHighlightedIndex);
      if (exampleObj && nextHighlightedIndex >= 0) {
        const fullPath = buildFullPath(allSuggestions[nextHighlightedIndex].suggestion);
        const value = getValueAtPath(exampleObj, fullPath);
        setHighlightedValue(value);
      } else {
        setHighlightedValue(undefined);
      }
      previousWordLength.current = word.length;
    }, [inputValue, flattenedData, isFocused, startWord, prefix, onChange, showValuePreview, exampleObj]);

    // Handle clicking outside to close suggestions
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
          setIsOpen(false);
          setIsFocused(false);
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
        !isInsideDoubleBraces(inputValue, cursorPosition)
      ) {
        const composedValue = replaceWordAtCursor(newValue, cursorPosition, `${prefix || ""}${suffix || ""}`);
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
    };

    const handleSuggestionClick = (suggestionItem: { suggestion: string; type: string }) => {
      const cursorPosition = inputRef.current?.selectionStart || 0;
      const { word, start } = getWordAtCursor(inputValue, cursorPosition);
      const normalizedWord = getNormalizedPath(word);
      const { basePath } = getPathContext(word, normalizedWord);
      const normalizedPath = suggestionItem.suggestion.startsWith(basePath)
        ? suggestionItem.suggestion
        : basePath
          ? `${basePath}.${suggestionItem.suggestion}`
          : suggestionItem.suggestion;
      const nextSuggestions = getAutocompleteSuggestions(flattenedData, normalizedPath);
      const nextSuggestionsAreArraySuggestions = nextSuggestions.some((suggestion: string) =>
        suggestion.match(/\[\d+\]$/),
      );
      const isObjectKey = nextSuggestions.length > 0 && !nextSuggestionsAreArraySuggestions;
      const displayPath = formatDisplayPath(parsePathSegments(normalizedPath), true);
      let newValue = displayPath;
      let cursorOffset = displayPath.length;
      if (isObjectKey) {
        newValue = `${displayPath}.`;
        cursorOffset = displayPath.length + 1;
      }

      newValue = replaceWordAtCursor(inputValue, cursorPosition, newValue);
      setInputValue(newValue);
      onChange?.(newValue);
      setHighlightedIndex(-1);
      requestAnimationFrame(() => {
        const cursorTarget = start + cursorOffset;
        inputRef.current?.setSelectionRange(cursorTarget, cursorTarget);
      });

      if (nextSuggestions.length === 0) {
        setIsOpen(false);
        return;
      }

      clearTimeout(blurTimeout);
      setTimeout(() => {
        setIsFocused(true);
        setIsOpen(true);
        setHighlightedIndex(highlightedIndex);
      }, 100);
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!isOpen || suggestions.length === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setHighlightedIndex((prev) => {
            const newIndex = prev < suggestions.length - 1 ? prev + 1 : 0;
            if (exampleObj && suggestions[newIndex]) {
              const fullPath = buildFullPath(suggestions[newIndex].suggestion);
              const value = getValueAtPath(exampleObj, fullPath);
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
              const fullPath = buildFullPath(suggestions[newIndex].suggestion);
              const value = getValueAtPath(exampleObj, fullPath);
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
            onFocus={() => {
              setIsFocused(true);
              if (suggestions.length > 0) {
                setIsOpen(true);
              }
            }}
            onBlur={() => {
              // Small delay to allow click on suggestions
              blurTimeout = setTimeout(() => {
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
                key={suggestionItem.suggestion}
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
                    const fullPath = buildFullPath(suggestionItem.suggestion);
                    const value = getValueAtPath(exampleObj, fullPath);
                    setHighlightedValue(value);
                  }
                }}
              >
                <span>{formatSuggestionLabel(suggestionItem.suggestion)}</span>
                <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">{suggestionItem.type}</span>
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
