import React, { useEffect, useRef, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

function isExpressionValue(value: unknown): boolean {
  return typeof value === "string" && value.includes("{{");
}

function fieldSupportsExpressions(field: FieldRendererProps["field"], allowExpressions: boolean): boolean {
  return allowExpressions && field.typeOptions?.select?.allowExpressions === true;
}

type ModeToggleProps = {
  testId: string;
  label: string;
  onClick: () => void;
};

const ModeToggle: React.FC<ModeToggleProps> = ({ testId, label, onClick }) => (
  <button
    type="button"
    className="text-xs text-muted-foreground hover:underline"
    data-testid={testId}
    onClick={onClick}
  >
    {label}
  </button>
);

const ExpressionInput: React.FC<FieldRendererProps & { onUseOptions: () => void }> = ({
  field,
  value,
  onChange,
  readOnly,
  autocompleteExampleObj,
  onUseOptions,
}) => (
  <div className="space-y-1">
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={(value as string) ?? ""}
      onChange={(nextValue) => onChange(nextValue || undefined)}
      placeholder={field.placeholder || "{{ expression }}"}
      startWord="{{"
      prefix="{{ "
      suffix=" }}"
      inputSize="md"
      showValuePreview
      quickTip="Tip: type `{{` to start an expression."
      className=""
      data-testid={toTestId(`field-${field.name}-expression`)}
    />
    {!readOnly && (
      <ModeToggle
        testId={toTestId(`field-${field.name}-use-options`)}
        label="Choose from options"
        onClick={onUseOptions}
      />
    )}
  </div>
);

const OptionsDropdown: React.FC<FieldRendererProps & { onUseExpression?: () => void }> = ({
  field,
  value,
  onChange,
  readOnly,
  onUseExpression,
}) => {
  const selectOptions = field.typeOptions?.select?.options ?? [];
  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  return (
    <div className="space-y-1">
      <Select
        value={(value as string) ?? (field.defaultValue as string) ?? ""}
        onValueChange={(val) => onChange(val || undefined)}
        disabled={readOnly}
      >
        <SelectTrigger className="w-full" data-testid={testId}>
          <SelectValue placeholder={`Select ${field.label || field.name}`} />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {selectOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value ?? ""} title={opt.description || undefined}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {onUseExpression && !readOnly && (
        <ModeToggle
          testId={toTestId(`field-${field.name}-use-expression`)}
          label="Use expression"
          onClick={onUseExpression}
        />
      )}
    </div>
  );
};

export const SelectFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  const { field, value, onChange, readOnly = false, allowExpressions = false } = props;
  const supportsExpressions = fieldSupportsExpressions(field, allowExpressions);
  const [expressionMode, setExpressionMode] = useState(() => supportsExpressions && isExpressionValue(value));
  const hasSetDefault = useRef(false);

  useEffect(() => {
    if (readOnly || expressionMode) return;

    if (!hasSetDefault.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
        hasSetDefault.current = true;
      }
    }
  }, [readOnly, expressionMode, value, field.defaultValue, onChange]);

  if (supportsExpressions && (expressionMode || isExpressionValue(value))) {
    return (
      <ExpressionInput
        {...props}
        onUseOptions={() => {
          setExpressionMode(false);
          onChange(undefined);
        }}
      />
    );
  }

  return (
    <OptionsDropdown {...props} onUseExpression={supportsExpressions ? () => setExpressionMode(true) : undefined} />
  );
};
