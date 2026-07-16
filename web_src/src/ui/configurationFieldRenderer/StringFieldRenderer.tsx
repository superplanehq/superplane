import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

function resolveStringFieldDisplayValue(value: unknown, readOnly: boolean, defaultValue: unknown): string {
  if (value !== undefined && value !== null) {
    return String(value);
  }

  if (readOnly && defaultValue !== undefined) {
    return String(defaultValue);
  }

  return "";
}

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  allowExpressions = false,
  excludedSuggestions,
  valuePreviewLabel,
  readOnly = false,
}) => {
  const hasInitialized = useRef(false);
  const shouldPreserveEmpty = field.togglable === true;

  // Set initial value only on first mount if no value is present but there's a default
  useEffect(() => {
    if (readOnly) {
      return;
    }

    if (!hasInitialized.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      hasInitialized.current = true;
      onChange(String(field.defaultValue));
    }
  }, [readOnly, value, field.defaultValue, onChange]);

  const currentValue = resolveStringFieldDisplayValue(value, readOnly, field.defaultValue);

  if (!allowExpressions) {
    return (
      <Input
        type={field.sensitive ? "password" : "text"}
        value={currentValue}
        onChange={(e) => {
          const nextValue = e.target.value;
          onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined);
        }}
        placeholder={field.placeholder || ""}
        className=""
        disabled={readOnly}
        data-testid={toTestId(`string-field-${field.name}`)}
      />
    );
  }

  if (field.sensitive) {
    return (
      <Input
        type="password"
        value={currentValue}
        onChange={(e) => {
          const nextValue = e.target.value;
          onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined);
        }}
        placeholder={field.placeholder || ""}
        className=""
        disabled={readOnly}
        data-testid={toTestId(`string-field-${field.name}`)}
      />
    );
  }

  return (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={(nextValue) => onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined)}
      placeholder={field.placeholder || ""}
      startWord="{{"
      prefix="{{ "
      suffix=" }}"
      inputSize="md"
      showValuePreview
      valuePreviewLabel={valuePreviewLabel}
      quickTip="Tip: type `{{` to start an expression."
      className=""
      disabled={readOnly}
      data-testid={toTestId(`string-field-${field.name}`)}
      excludedSuggestions={excludedSuggestions}
    />
  );
};
