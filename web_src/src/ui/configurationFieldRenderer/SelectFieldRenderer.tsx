import { createPortal } from "react-dom";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toTestId } from "@/lib/testID";
import React, { useEffect, useRef, useState } from "react";
import { isExpressionValue } from "./expressionValue";
import type { FieldRendererProps } from "./types";

export const SelectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  allowExpressions = false,
  autocompleteExampleObj,
  labelRightRef,
  labelRightReady = false,
}) => {
  const selectOptions = field.typeOptions?.select?.options ?? [];
  const hasSetDefault = useRef(false);
  const initialIsExpression = allowExpressions && isExpressionValue(value);
  const [useExpressionMode, setUseExpressionMode] = useState(initialIsExpression);

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
  const currentValue = (value as string) ?? "";

  const selectControl = (
    <Select
      value={currentValue || (field.defaultValue as string) || ""}
      onValueChange={(val) => onChange(val || undefined)}
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
  );

  if (!allowExpressions) {
    return selectControl;
  }

  const expressionInput = (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={(nextValue) => onChange(nextValue || undefined)}
      placeholder={field.placeholder || `e.g. {{ previous().data.status == 'ok' ? 'green' : 'red' }}`}
      startWord="{{"
      prefix="{{ "
      suffix=" }}"
      inputSize="md"
      showValuePreview
      quickTip="Tip: type `{{` to start an expression. Result must resolve to a supported option value."
      className=""
      data-testid={field.name ? toTestId(`field-${field.name}-expression`) : undefined}
    />
  );

  const tabsList = (
    <TabsList className="h-7 rounded-md p-0.5">
      <TabsTrigger value="fixed" className="text-xs px-2 py-1 data-[state=active]:shadow-sm">
        Fixed
      </TabsTrigger>
      <TabsTrigger value="expression" className="text-xs px-2 py-1 data-[state=active]:shadow-sm">
        Expression
      </TabsTrigger>
    </TabsList>
  );
  const tabsInLabelRow =
    labelRightReady && labelRightRef?.current ? createPortal(tabsList, labelRightRef.current) : null;

  const handleTabChange = (next: string) => {
    const nextExpression = next === "expression";
    if (nextExpression !== useExpressionMode) {
      onChange(undefined);
    }
    setUseExpressionMode(nextExpression);
  };

  return (
    <Tabs value={useExpressionMode ? "expression" : "fixed"} onValueChange={handleTabChange}>
      {tabsInLabelRow ?? <div className="flex justify-end">{tabsList}</div>}
      <TabsContent value="fixed">{selectControl}</TabsContent>
      <TabsContent value="expression">{expressionInput}</TabsContent>
    </Tabs>
  );
};
