import React, { useState, useEffect, useRef, forwardRef } from 'react';
import { twMerge } from 'tailwind-merge';
import { flattenForAutocomplete, getAutocompleteSuggestions } from './core';

export interface AutoCompleteInputProps extends Omit<React.ComponentPropsWithoutRef<'input'>, 'onChange'> {
  exampleObj: Record<string, unknown>;
  value?: string;
  onChange?: (value: string) => void;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
}

let blurTimeout: NodeJS.Timeout;

export const AutoCompleteInput = forwardRef<HTMLInputElement, AutoCompleteInputProps>(
  ({ exampleObj, value = '', onChange, className, placeholder = 'Type to search...', disabled, ...props }) => {
    const [inputValue, setInputValue] = useState(value);
    const [suggestions, setSuggestions] = useState<string[]>([]);
    const [isOpen, setIsOpen] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    const [highlightedIndex, setHighlightedIndex] = useState(-1);
    const [flattenedData, setFlattenedData] = useState<Record<string, string[]>>({});

    const containerRef = useRef<HTMLDivElement>(null);
    const suggestionsRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

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
      const lastKey = inputValue.split('.').slice(-1)[0];
      if (flattenedData) {
        const parsedInput = inputValue.split('.').slice(0, -1).join('.');
        const newSuggestions = getAutocompleteSuggestions(flattenedData, parsedInput || 'root');
        const arraySuggestions = getAutocompleteSuggestions(flattenedData, parsedInput ? `${parsedInput}.${lastKey}` : lastKey).filter((suggestion: string) => suggestion.match(/\[\d+\]$/));
        const similarSuggestions = newSuggestions.filter((suggestion: string) => suggestion.startsWith(lastKey) && suggestion !== lastKey);
        const allSuggestions = [...new Set([...arraySuggestions, ...similarSuggestions])];
        setSuggestions(allSuggestions);
        setIsOpen(isFocused && allSuggestions.length > 0);
        setHighlightedIndex(-1);
      } else {
        setSuggestions([]);
        setIsOpen(false);
      }
    }, [inputValue, flattenedData, isFocused]);

    // Handle clicking outside to close suggestions
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
          setIsOpen(false);
          setIsFocused(false);
          setHighlightedIndex(-1);
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setInputValue(newValue);
      onChange?.(newValue);
    };

    const handleSuggestionClick = (suggestion: string) => {
      const allPreviousKeys = inputValue.split('.');
      const withoutLastKey = allPreviousKeys.slice(0, -1).join('.');
      let newValue = suggestion.startsWith(withoutLastKey) ? suggestion : `${withoutLastKey}.${suggestion}`;
      const nextSuggestions = getAutocompleteSuggestions(flattenedData, newValue);
      const nextSuggestionsAreArraySuggestions = nextSuggestions.some((suggestion: string) => suggestion.match(/\[\d+\]$/));
      newValue += (nextSuggestions.length > 0 && !nextSuggestionsAreArraySuggestions) ? '.' : '';
      setInputValue(newValue);
      onChange?.(newValue);
      setHighlightedIndex(-1);

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
        case 'ArrowDown':
          e.preventDefault();
          setHighlightedIndex(prev =>
            prev < suggestions.length - 1 ? prev + 1 : 0
          );
          break;
        case 'ArrowUp':
          e.preventDefault();
          setHighlightedIndex(prev =>
            prev > 0 ? prev - 1 : suggestions.length - 1
          );
          break;
        case 'Enter':
          e.preventDefault();
          if (highlightedIndex >= 0) {
            handleSuggestionClick(suggestions[highlightedIndex]);
          }
          break;
        case 'Escape':
          setIsOpen(false);
          setHighlightedIndex(-1);
          break;
      }
    };

    // Scroll highlighted item into view
    useEffect(() => {
      if (highlightedIndex >= 0 && suggestionsRef.current) {
        const highlightedElement = suggestionsRef.current.children[highlightedIndex] as HTMLElement;
        if (highlightedElement) {
          highlightedElement.scrollIntoView({
            block: 'nearest',
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
            'relative block w-full',
            'before:absolute before:inset-px before:rounded-[calc(var(--radius-lg)-1px)] before:bg-white before:shadow-sm',
            'dark:before:hidden',
            'after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset sm:focus-within:after:ring-2 sm:focus-within:after:ring-blue-500',
            'has-data-disabled:opacity-50 has-data-disabled:before:bg-zinc-950/5 has-data-disabled:before:shadow-none',
            'has-data-invalid:before:shadow-red-500/10',
            className
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
              }, 150);
            }}
            placeholder={placeholder}
            disabled={disabled}
            className={twMerge([
              'relative block w-full appearance-none rounded-lg px-3 py-2 sm:px-3 sm:py-1.5',
              'text-base/6 text-zinc-950 placeholder:text-zinc-500 sm:text-sm/6 dark:text-white',
              'border border-zinc-950/10 hover:border-zinc-950/20 dark:border-white/10 dark:hover:border-white/20',
              'bg-transparent dark:bg-white/5',
              'focus:outline-none',
              'invalid:border-red-500 dark:invalid:border-red-500',
              'disabled:border-zinc-950/20 dark:disabled:border-white/15 dark:disabled:bg-white/2.5',
            ])}
            {...props}
          />
        </span>

        {/* Suggestions Dropdown */}
        {isOpen && suggestions.length > 0 && (
          <div
            ref={suggestionsRef}
            className={twMerge([
              'absolute z-50 w-full mt-1 bg-white border border-zinc-200 rounded-lg shadow-lg max-h-60 overflow-auto',
              'dark:bg-zinc-800 dark:border-zinc-700'
            ])}
          >
            {suggestions.map((suggestion, index) => (
              <div
                key={suggestion}
                className={twMerge([
                  'px-3 py-2 cursor-pointer text-sm',
                  'hover:bg-zinc-100 dark:hover:bg-zinc-700',
                  'text-zinc-950 dark:text-white',
                  highlightedIndex === index && 'bg-zinc-100 dark:bg-zinc-700'
                ])}
                onClick={() => handleSuggestionClick(suggestion)}
                onMouseEnter={() => setHighlightedIndex(index)}
              >
                {suggestion}
              </div>
            ))}
          </div>
        )}

        {/* Empty State */}
        {isOpen && suggestions.length === 0 && inputValue && (
          <div className={twMerge([
            'absolute z-50 w-full mt-1 bg-white border border-zinc-200 rounded-lg shadow-lg',
            'dark:bg-zinc-800 dark:border-zinc-700'
          ])}>
            <div className="px-3 py-2 text-sm text-zinc-500 dark:text-zinc-400">
              No results found.
            </div>
          </div>
        )}
      </div>
    );
  }
);