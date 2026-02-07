import { useState, useRef, useEffect } from "react";
import { useFloating, autoUpdate, offset, flip, shift, size } from "@floating-ui/react";
import { Icon } from "@/components/Icon";
import { twMerge } from "tailwind-merge";

export interface AutoCompleteOption {
  value: string;
  label: string;
  group?: string;
  type?: string;
}

export interface AutoCompleteSelectProps {
  options: AutoCompleteOption[];
  value?: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  error?: boolean;
  disabled?: boolean;
}

export function AutoCompleteSelect({
  options,
  value = "",
  onChange,
  placeholder = "Search...",
  className,
  error = false,
  disabled = false,
}: AutoCompleteSelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const { refs, floatingStyles } = useFloating({
    open: isOpen,
    onOpenChange: setIsOpen,
    middleware: [
      offset(4),
      flip(),
      shift(),
      size({
        apply({ rects, elements }) {
          Object.assign(elements.floating.style, {
            minWidth: `${rects.reference.width}px`,
          });
        },
      }),
    ],
    whileElementsMounted: autoUpdate,
  });

  // Find the selected option
  const selectedOption = options.find((option) => option.value === value);

  // Filter options based on query
  const filteredOptions =
    query === ""
      ? options
      : options.filter(
          (option) =>
            option.label.toLowerCase().includes(query.toLowerCase()) ||
            option.value.toLowerCase().includes(query.toLowerCase()),
        );

  // Group filtered options
  const groupedOptions: Record<string, AutoCompleteOption[]> = {};
  filteredOptions.forEach((option) => {
    const group = option.group || "Options";
    if (!groupedOptions[group]) {
      groupedOptions[group] = [];
    }
    groupedOptions[group].push(option);
  });

  const handleInputFocus = () => {
    setIsOpen(true);
    setQuery("");
  };

  const handleInputBlur = (e: React.FocusEvent) => {
    if (listRef.current?.contains(e.relatedTarget as Node)) {
      return;
    }
    setTimeout(() => {
      setQuery("");
      setIsOpen(false);
    }, 150);
  };

  const handleOptionSelect = (optionValue: string) => {
    onChange(optionValue);
    setTimeout(() => {
      inputRef.current?.blur();
      setQuery("");
      setIsOpen(false);
    }, 150);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value);
    if (!isOpen) setIsOpen(true);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setIsOpen(false);
      setQuery("");
      inputRef.current?.blur();
    }
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const referenceEl = refs.reference.current;
      const floatingEl = refs.floating.current;

      if (
        referenceEl &&
        floatingEl &&
        referenceEl instanceof Element &&
        floatingEl instanceof Element &&
        !referenceEl.contains(event.target as Node) &&
        !floatingEl.contains(event.target as Node)
      ) {
        setIsOpen(false);
        setQuery("");
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [refs.reference, refs.floating]);

  return (
    <div className="relative">
      <div
        ref={refs.setReference}
        className={twMerge(
          "relative flex items-center w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-100",
          "border rounded-md focus-within:outline-none focus-within:ring-2 cursor-pointer",
          error
            ? "border-red-300 dark:border-red-600 focus-within:ring-red-500"
            : "border-gray-300 dark:border-gray-600 focus-within:ring-blue-500",
          disabled && "opacity-50 cursor-not-allowed",
          className,
        )}
        onClick={() => {
          if (!isOpen) {
            setIsOpen(true);
            setQuery("");
          }
          inputRef.current?.focus();
        }}
      >
        {!isOpen && selectedOption && query === "" ? (
          <span className="flex-1 text-gray-800 dark:text-gray-100 truncate">{selectedOption.label}</span>
        ) : (
          <input
            ref={inputRef}
            type="text"
            role="combobox"
            aria-expanded={isOpen}
            aria-haspopup="listbox"
            className="flex-1 bg-transparent border-none outline-none placeholder:text-gray-500 dark:placeholder:text-gray-400"
            placeholder={placeholder}
            value={query}
            onChange={handleInputChange}
            onFocus={handleInputFocus}
            onBlur={handleInputBlur}
            onKeyDown={handleKeyDown}
            disabled={disabled}
          />
        )}
        <div
          className="ml-2"
          onClick={(e) => {
            e.stopPropagation();
            setIsOpen(!isOpen);
          }}
        >
          <Icon
            name="chevron-down"
            size="sm"
            className={twMerge(
              "ml-2 text-gray-400 dark:text-gray-500 flex-shrink-0 transition-transform",
              isOpen && "rotate-180",
            )}
          />
        </div>
      </div>

      {isOpen && (
        <div
          ref={refs.setFloating}
          style={floatingStyles}
          role="listbox"
          className="z-50 max-h-60 overflow-auto rounded-md bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 focus:outline-none"
        >
          <div ref={listRef}>
            {filteredOptions.length === 0 ? (
              <div className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
                {query !== "" ? "No options found" : "No connections available"}
              </div>
            ) : (
              Object.entries(groupedOptions).map(([groupName, groupOptions]) => (
                <div key={groupName}>
                  {Object.keys(groupedOptions).length > 1 && (
                    <div className="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/50 border-b border-gray-200 dark:border-gray-700">
                      {groupName}
                    </div>
                  )}
                  {groupOptions.map((option) => {
                    const isSelected = option.value === value;
                    return (
                      <div
                        key={option.value}
                        role="option"
                        aria-selected={isSelected}
                        className="relative cursor-pointer select-none px-3 py-2 text-sm hover:bg-blue-500 hover:text-white text-gray-800 dark:text-gray-100"
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => handleOptionSelect(option.value)}
                      >
                        <div className="flex items-center justify-between">
                          <span className={twMerge("block truncate", isSelected ? "font-medium" : "font-normal")}>
                            {option.label}
                          </span>
                          {isSelected && <Icon name="check" size="sm" className="text-blue-500" />}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
