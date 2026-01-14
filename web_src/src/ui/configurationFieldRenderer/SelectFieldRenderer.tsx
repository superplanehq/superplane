import React, { useEffect, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

// Helper function to determine if a when_to_run option is Block or Allow
const getWhenToRunType = (value: string | undefined): "block" | "allow" | null => {
  if (!value) return null;

  // Block logic options
  if (value === "custom_exclude" || value === "template_no_weekends") {
    return "block";
  }

  // Allow logic options
  if (
    value === "custom_include" ||
    value === "template_working_hours" ||
    value === "template_outside_working_hours" ||
    value === "template_weekends"
  ) {
    return "allow";
  }

  return null;
};

export const SelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const selectOptions = field.typeOptions?.select?.options ?? [];
  const hasSetDefault = useRef(false);
  const isWhenToRunField = field.name === "when_to_run";

  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
        hasSetDefault.current = true;
      }
    }
  }, [value, field.defaultValue, onChange]);

  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  return (
    <Select
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`} data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || field.name}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {selectOptions.map((opt) => {
          const optionType = isWhenToRunField ? getWhenToRunType(opt.value as string) : null;
          return (
            <SelectItem key={opt.value} value={opt.value ?? ""}>
              <span className="flex items-center gap-2">
                {optionType && (
                  <span
                    className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium ${
                      optionType === "block"
                        ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                        : "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                    }`}
                  >
                    {optionType === "block" ? "Block" : "Allow"}
                  </span>
                )}
                {opt.label}
              </span>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
};
