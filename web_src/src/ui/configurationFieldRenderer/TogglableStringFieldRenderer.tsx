import React from "react";
import { Input } from "../input";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";
import { ConfigurationTypeOptions } from "@/api-client";

export const TogglableStringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  // Determine if the field is enabled (has a non-null value)
  const isEnabled = value !== null && value !== undefined;

  // Get the string value (empty string if disabled)
  const stringValue = isEnabled ? (value as string) || "" : "";

  // Get placeholder from field type options or fallback to field placeholder
  // Note: Using any cast until API types are regenerated to include togglableString
  const placeholder =
    (field.typeOptions as ConfigurationTypeOptions)?.togglableString?.placeholder || field.placeholder || "";

  const handleToggleChange = (checked: boolean) => {
    if (checked) {
      // Enable the field with empty string
      onChange("");
    } else {
      // Disable the field by setting to null
      onChange(null);
    }
  };

  const handleStringChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    // Only update if field is enabled
    if (isEnabled) {
      onChange(newValue || undefined);
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
        <Input
          type={field.sensitive ? "password" : "text"}
          value={stringValue}
          onChange={handleStringChange}
          placeholder={placeholder}
          disabled={!isEnabled}
          className={`${hasError ? "border-red-500 border-2" : ""} ${!isEnabled ? "opacity-50" : ""}`}
        />
      </div>
    </div>
  );
};
