import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { ExpressionEditor } from "@/components/ExpressionEditor";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  allowExpressions = false,
  excludedSuggestions,
  valuePreviewLabel,
}) => {
  const hasInitialized = useRef(false);
  const shouldPreserveEmpty = field.togglable === true;

  // Set initial value only on first mount if no value is present but there's a default
  useEffect(() => {
    if (!hasInitialized.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      hasInitialized.current = true;
      onChange(String(field.defaultValue));
    }
  }, [value, field.defaultValue, onChange]);

  const currentValue = (value as string) ?? "";

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
        data-testid={toTestId(`string-field-${field.name}`)}
      />
    );
  }

  return (
    <ExpressionEditor
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={(nextValue) => onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined)}
      placeholder={field.placeholder || ""}
      inputSize="md"
      showValuePreview
      valuePreviewLabel={valuePreviewLabel}
      className=""
      data-testid={toTestId(`string-field-${field.name}`)}
      excludedSuggestions={excludedSuggestions}
    />
  );
};
