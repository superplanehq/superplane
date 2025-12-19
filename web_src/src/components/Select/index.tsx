import { useState, useRef, useEffect } from "react";
import { useFloating, autoUpdate, offset, flip, shift, size } from "@floating-ui/react";
import { Icon } from "@/components/Icon";
import { twMerge } from "tailwind-merge";

export interface SelectOption {
  value: string;
  label: string;
  description?: string;
  icon?: string;
  image?: string;
}

export interface SelectProps {
  options: SelectOption[];
  value?: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  error?: boolean;
  disabled?: boolean;
}

export function Select({
  options,
  value = "",
  onChange,
  placeholder = "Select an option...",
  className,
  error = false,
  disabled = false,
}: SelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const triggerRef = useRef<HTMLDivElement>(null);
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

  const handleTriggerClick = () => {
    if (!disabled) {
      setIsOpen(!isOpen);
    }
  };

  const handleOptionSelect = (optionValue: string) => {
    onChange(optionValue);
    setIsOpen(false);
    triggerRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (disabled) return;

    switch (e.key) {
      case "Enter":
      case " ":
        e.preventDefault();
        setIsOpen(!isOpen);
        break;
      case "Escape":
        setIsOpen(false);
        triggerRef.current?.focus();
        break;
      case "ArrowDown":
        e.preventDefault();
        if (!isOpen) {
          setIsOpen(true);
        } else {
          // Focus first option or next option
          const currentIndex = options.findIndex((opt) => opt.value === value);
          const nextIndex = currentIndex < options.length - 1 ? currentIndex + 1 : 0;
          onChange(options[nextIndex].value);
        }
        break;
      case "ArrowUp":
        e.preventDefault();
        if (!isOpen) {
          setIsOpen(true);
        } else {
          // Focus last option or previous option
          const currentIndex = options.findIndex((opt) => opt.value === value);
          const prevIndex = currentIndex > 0 ? currentIndex - 1 : options.length - 1;
          onChange(options[prevIndex].value);
        }
        break;
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
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [refs.reference, refs.floating]);

  return (
    <div className="relative">
      <div
        ref={refs.setReference}
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-disabled={disabled}
        onClick={handleTriggerClick}
        onKeyDown={handleKeyDown}
        className={twMerge(
          "relative flex items-center justify-between w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100",
          "border rounded-md cursor-pointer focus:outline-none focus:ring-2",
          error
            ? "border-red-300 dark:border-red-600 focus:ring-red-500"
            : "border-gray-300 dark:border-gray-600 focus:ring-blue-500",
          disabled && "opacity-50 cursor-not-allowed",
          className,
        )}
      >
        <div className="flex items-center min-w-0 flex-1">
          {(selectedOption?.image || selectedOption?.icon) && (
            <div className="w-4 h-4 mr-2 flex-shrink-0 flex items-center justify-center">
              {selectedOption?.image && (
                <img src={selectedOption.image} alt="" className="w-4 h-4 dark:bg-white dark:rounded-sm dark:p-0.5" />
              )}
              {selectedOption?.icon && !selectedOption?.image && <Icon name={selectedOption.icon} size="sm" />}
            </div>
          )}
          <span className={twMerge("block truncate", !selectedOption && "text-gray-500 dark:text-gray-400")}>
            {selectedOption ? selectedOption.label : placeholder}
          </span>
        </div>
        <Icon
          name={isOpen ? "expand_less" : "expand_more"}
          size="sm"
          className="ml-2 text-gray-400 dark:text-gray-500 flex-shrink-0"
        />
      </div>

      {isOpen && (
        <div
          ref={refs.setFloating}
          style={floatingStyles}
          role="listbox"
          className="z-50 max-h-60 overflow-auto rounded-md bg-white dark:bg-gray-800 shadow-lg border border-gray-200 dark:border-gray-700 focus:outline-none"
        >
          <div ref={listRef}>
            {options.length === 0 ? (
              <div className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">No options available</div>
            ) : (
              options.map((option) => {
                const isSelected = option.value === value;
                return (
                  <div
                    key={option.value}
                    role="option"
                    aria-selected={isSelected}
                    className="relative cursor-pointer select-none px-3 py-2 text-sm hover:bg-blue-50 dark:hover:bg-blue-900/20 text-gray-900 dark:text-gray-100 transition-colors duration-150"
                    onClick={(e) => {
                      e.preventDefault();
                      handleOptionSelect(option.value);
                    }}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center min-w-0 flex-1">
                        {(option.image || option.icon) && (
                          <div className="w-4 h-4 mr-2 flex-shrink-0 flex items-center justify-center">
                            {option.image && (
                              <img
                                src={option.image}
                                alt=""
                                className="w-4 h-4 dark:bg-white dark:rounded-sm dark:p-0.5"
                              />
                            )}
                            {option.icon && !option.image && <Icon name={option.icon} size="sm" />}
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <span className={twMerge("block truncate", isSelected ? "font-medium" : "font-normal")}>
                            {option.label}
                          </span>
                          {option.description && (
                            <span className="block truncate text-xs text-gray-500 dark:text-gray-400">
                              {option.description}
                            </span>
                          )}
                        </div>
                      </div>
                      {isSelected && <Icon name="check" size="sm" className="text-blue-500 ml-2 flex-shrink-0" />}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}
