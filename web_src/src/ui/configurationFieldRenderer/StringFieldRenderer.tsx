import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { InlineFieldAssistant } from "@/ui/InlineFieldAssistant";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  allowExpressions = false,
  suggestFieldValue,
  assistantEnabled = false,
  labelRightRef,
  labelRightReady = false,
}) => {
  const hasInitialized = useRef(false);
  const shouldPreserveEmpty = field.togglable === true;
  const showAssistant = Boolean(assistantEnabled && suggestFieldValue && !field.sensitive && field.type === "string");
  const fieldLabel = field.label || field.name || "Field";

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
      <div className="space-y-2">
        {showAssistant ? (
          <InlineFieldAssistant
            fieldLabel={fieldLabel}
            onApplyValue={(next) => {
              onChange(shouldPreserveEmpty ? next : next || undefined);
            }}
            suggestFieldValue={suggestFieldValue}
            assistantEnabled={assistantEnabled}
            labelRightRef={labelRightRef}
            labelRightReady={labelRightReady}
          />
        ) : null}
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
      </div>
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
    <div className="space-y-2">
      {showAssistant ? (
        <InlineFieldAssistant
          fieldLabel={fieldLabel}
          onApplyValue={(next) => {
            onChange(shouldPreserveEmpty ? next : next || undefined);
          }}
          suggestFieldValue={suggestFieldValue}
          assistantEnabled={assistantEnabled}
          labelRightRef={labelRightRef}
          labelRightReady={labelRightReady}
        />
      ) : null}
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
    </div>
  );
};
