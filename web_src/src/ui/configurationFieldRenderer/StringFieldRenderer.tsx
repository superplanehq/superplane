import React from "react";
import { Input } from "../input";
import { FieldRendererProps } from "./types";
import { cn } from "@/lib/utils";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Input
      type={field.sensitive ? "password" : "text"}
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.placeholder || ""}
      className={cn(
        hasError ? "border-red-500 border-2" : "",
        field.readOnly ? "bg-gray-50 dark:bg-gray-800/50 cursor-not-allowed" : "",
      )}
      readOnly={field.readOnly}
      disabled={field.readOnly}
    />
  );
};
