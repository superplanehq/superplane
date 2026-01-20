import React from "react";
import { Input } from "@/components/ui/input";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const ExpressionFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
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
        className=""
        data-testid={toTestId(`expression-field-${field.name}`)}
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
        className=""
        data-testid={toTestId(`expression-field-${field.name}`)}
      />
    );
  }

  return (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={(nextValue) => onChange(nextValue || undefined)}
      placeholder={field.placeholder || ""}
      inputSize="md"
      showValuePreview
      quickTip="Tip: type `$` to browse node outputs."
      className=""
      data-testid={toTestId(`expression-field-${field.name}`)}
    />
  );
};
