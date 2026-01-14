import React, { useEffect, useRef } from "react";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

export const RadioButtonFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
}) => {
  const selectOptions = field.typeOptions?.select?.options ?? [];
  const hasSetDefault = useRef(false);
  const currentValue = (value as string) ?? (field.defaultValue as string) ?? "";

  useEffect(() => {
    if (!hasSetDefault.current && (value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
        hasSetDefault.current = true;
      }
    }
  }, [value, field.defaultValue, onChange]);

  const testId = field.name ? toTestId(`field-${field.name}-tabs`) : undefined;

  // If no value is set and we have options, set the first one as default
  const activeValue = currentValue || selectOptions[0]?.value || "";

  return (
    <div className={hasError ? "border-red-500 border-2 rounded p-2" : ""}>
      <Tabs
        value={activeValue}
        onValueChange={(val) => onChange(val)}
        className="w-full"
      >
        <TabsList className="grid w-full grid-cols-2">
          {selectOptions.map((opt) => {
            const optionValue = opt.value ?? "";
            return (
              <TabsTrigger
                key={optionValue}
                value={optionValue}
                data-testid={testId ? `${testId}-${optionValue}` : undefined}
              >
                {opt.label}
              </TabsTrigger>
            );
          })}
        </TabsList>
      </Tabs>
    </div>
  );
};
