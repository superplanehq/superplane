import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";
import { ConfigurationSelectOption } from "@/api-client";

export const TogglableSelectFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  // Determine if the field is enabled (has a non-null value)
  const isEnabled = value !== null && value !== undefined;

  // Get the string value (empty string if disabled)
  const selectValue = isEnabled ? (value as string) || "" : "";

  // Get options from field type options
  const selectOptions = field.typeOptions?.togglableSelect?.options || [];

  const handleToggleChange = (checked: boolean) => {
    if (checked) {
      // Enable the field with the first option or empty string
      const defaultOption = selectOptions.length > 0 ? selectOptions[0].value : "";
      onChange(defaultOption);
    } else {
      // Disable the field by setting to null
      onChange(null);
    }
  };

  const handleSelectChange = (val: string) => {
    // Only update if field is enabled
    if (isEnabled) {
      onChange(val || undefined);
    }
  };

  return (
    <div className="flex items-center gap-3">
      <Switch
        checked={isEnabled}
        onCheckedChange={handleToggleChange}
        className={hasError ? "border-red-500 border-2" : ""}
      />
      <div className="flex-1">
        <Select value={selectValue} onValueChange={handleSelectChange} disabled={!isEnabled}>
          <SelectTrigger
            className={`w-full ${hasError ? "border-red-500 border-2" : ""} ${!isEnabled ? "opacity-50" : ""}`}
          >
            <SelectValue placeholder={`Select ${field.label || field.name}`} />
          </SelectTrigger>
          <SelectContent className="max-h-60">
            {selectOptions.map((opt: ConfigurationSelectOption) => (
              <SelectItem key={opt.value} value={opt.value ?? ""}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
};
