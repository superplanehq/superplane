import React from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const StringFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Input
      type={field.sensitive ? "password" : "text"}
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.placeholder || ""}
      className={hasError ? "border-red-500 border-2" : ""}
      data-testid={toTestId(`string-field-${field.name}`)}
    />
  );
};
