import React from "react";
import { FieldRendererProps } from "./types";

export const BooleanFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <input
      type="checkbox"
      checked={(value as boolean) ?? field.defaultValue === "true" ?? false}
      onChange={(e) => onChange(e.target.checked)}
      className={`h-4 w-4 rounded ${hasError ? "border-red-500 border-2" : "border-gray-300 dark:border-gray-700"}`}
    />
  );
};
