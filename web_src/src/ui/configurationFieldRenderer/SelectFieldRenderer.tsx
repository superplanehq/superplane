import React, { useEffect, useRef } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";

export const SelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const selectOptions = field.typeOptions?.select?.options ?? [];
  const hasSetDefault = useRef(false);

  const isTogglable = field.togglable === true;

  const isEnabled = isTogglable ? value !== null && value !== undefined : true;

  const selectValue = isEnabled ? ((value as string) ?? (field.defaultValue as string) ?? "") : "";

  const handleToggleChange = (checked: boolean) => {
    if (!isTogglable) return;

    if (checked) {
      const defaultVal = field.defaultValue as string;
      const initialValue =
        defaultVal && selectOptions.some((opt) => opt.value === defaultVal)
          ? defaultVal
          : selectOptions.length > 0
            ? selectOptions[0].value
            : "";
      onChange(initialValue);
    } else {
      onChange(null);
    }
  };

  useEffect(() => {
    if (
      !hasSetDefault.current &&
      isEnabled &&
      (value === undefined || value === null) &&
      field.defaultValue !== undefined
    ) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
        hasSetDefault.current = true;
      }
    }
  }, [value, field.defaultValue, onChange, isEnabled]);

  if (isTogglable) {
    return (
      <div className="flex items-center gap-3">
        <Switch
          checked={isEnabled}
          onCheckedChange={handleToggleChange}
          className={`${hasError ? "border-red-500 border-2" : ""}`}
        />
        <div className={`flex-1 ${!isEnabled ? "opacity-50" : ""}`}>
          <Select
            value={selectValue}
            onValueChange={(val) => isEnabled && onChange(val || undefined)}
            disabled={!isEnabled}
          >
            <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`}>
              <SelectValue placeholder={`Select ${field.label || field.name}`} />
            </SelectTrigger>
            <SelectContent className="max-h-60">
              {selectOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value ?? ""}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    );
  }

  return (
    <Select
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`}>
        <SelectValue placeholder={`Select ${field.label || field.name}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        {selectOptions.map((opt) => (
          <SelectItem key={opt.value} value={opt.value ?? ""}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
