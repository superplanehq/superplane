import { Checkbox } from "@/ui/checkbox";
import { Switch } from "@/ui/switch";
import React from "react";
import type { FieldRendererProps } from "./types";

export const BooleanFieldRenderer: React.FC<FieldRendererProps & { labeled?: boolean }> = ({
  field,
  value,
  onChange,
  readOnly = false,
  labeled = false,
}) => {
  const fieldName = field?.name || "";
  const inputId = React.useId();

  const checked = React.useMemo(
    () => coerceFieldValuesIntoBoolean(fieldName, value, field?.defaultValue),
    [fieldName, value, field?.defaultValue],
  );

  if (labeled) {
    const valueLabel = checked ? "True" : "False";

    return (
      <label htmlFor={inputId} className="contents cursor-pointer">
        <div className="flex w-9 shrink-0 items-center justify-end">
          <Checkbox
            id={inputId}
            checked={checked}
            disabled={readOnly}
            onCheckedChange={(next) => onChange(next === true)}
          />
        </div>
        <span className="text-sm text-gray-600 dark:text-gray-400">{valueLabel}</span>
      </label>
    );
  }

  return <Switch checked={checked} onCheckedChange={onChange} disabled={readOnly} className="" />;
};

function coerceFieldValuesIntoBoolean(
  fieldName: string,
  value: FieldRendererProps["value"],
  defaultValue: string | undefined,
): boolean {
  // if the value is a boolean, return it
  if (typeof value === "boolean") {
    return value;
  }

  // if the value is a string, parse it as a boolean
  if (typeof value === "string") {
    if (value === "true") {
      return true;
    }

    if (value === "false") {
      return false;
    }

    // invalid string boolean value. log a warning and return false.
    console.warn(`Invalid boolean value: ${value} for field ${fieldName}. Returning false.`);
    return false;
  }

  // if the value is undefined, return the default value
  if (value === undefined) {
    return defaultValue === "true";
  }

  // if the value is not a boolean or string, log a warning and return false.
  console.warn(`Invalid boolean value: ${value} for field ${fieldName}. Returning false. Expected boolean or string.`);
  return false;
}
