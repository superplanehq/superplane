import React from "react";
import { Input } from "../input";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const isTogglable = field.togglable === true;
  const isEnabled = isTogglable ? value !== null && value !== undefined : true;
  const inputValue = isEnabled ? ((value as string) ?? (field.defaultValue as string) ?? "") : "";

  const handleToggleChange = (checked: boolean) => {
    if (!isTogglable) return;

    if (checked) {
      const defaultVal = field.defaultValue as string;
      onChange(defaultVal || "");
    } else {
      onChange(null);
    }
  };

  if (isTogglable) {
    return (
      <div className="flex items-center gap-3">
        <Switch
          checked={isEnabled}
          onCheckedChange={handleToggleChange}
          className={`${hasError ? "border-red-500 border-2" : ""}`}
        />
        <div className={`flex-1 ${!isEnabled ? "opacity-50" : ""}`}>
          <Input
            type={field.sensitive ? "password" : "text"}
            value={inputValue}
            onChange={(e) => isEnabled && onChange(e.target.value || undefined)}
            placeholder={field.placeholder || ""}
            className={hasError ? "border-red-500 border-2" : ""}
            disabled={!isEnabled}
          />
        </div>
      </div>
    );
  }

  return (
    <Input
      type={field.sensitive ? "password" : "text"}
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.placeholder || ""}
      className={hasError ? "border-red-500 border-2" : ""}
    />
  );
};
