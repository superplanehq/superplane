import React from "react";
import { Input } from "@/components/ui/input";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  autocompleteExampleObj,
}) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";

  if (field.disallowExpression) {
    return (
      <Input
        type={field.sensitive ? "password" : "text"}
        value={currentValue}
        onChange={(e) => onChange(e.target.value || undefined)}
        placeholder={field.placeholder || ""}
        className={hasError ? "border-red-500 border-2" : ""}
        data-testid={toTestId(`string-field-${field.name}`)}
      />
    );
  }

  if (field.sensitive) {
    return (
      <Input
        type="password"
        value={currentValue}
        onChange={(e) => onChange(e.target.value || undefined)}
        placeholder={field.placeholder || ""}
        className={hasError ? "border-red-500 border-2" : ""}
        data-testid={toTestId(`string-field-${field.name}`)}
      />
    );
  }

  return (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={(nextValue) => onChange(nextValue || undefined)}
      placeholder={field.placeholder || ""}
      startWord="{{"
      prefix="{{ $"
      suffix=" }}"
      inputSize="md"
      showValuePreview
      className={hasError ? "after:ring-2 after:ring-red-500" : ""}
      data-testid={toTestId(`string-field-${field.name}`)}
    />
  );
};
