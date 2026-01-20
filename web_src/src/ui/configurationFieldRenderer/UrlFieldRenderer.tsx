import React from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";

export const UrlFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Input
      type="url"
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.placeholder || `https://example.com`}
      className=""
    />
  );
};
