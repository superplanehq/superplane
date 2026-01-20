import React from "react";
import { Input } from "@/components/ui/input";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
}) => {
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";
  const shouldPreserveEmpty = field.togglable === true;

  if (field.disallowExpression) {
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
      quickTip="Tip: type `{{` to start an expression."
      className=""
      data-testid={toTestId(`string-field-${field.name}`)}
    />
  );
};
