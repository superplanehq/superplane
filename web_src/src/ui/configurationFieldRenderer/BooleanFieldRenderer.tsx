import React from "react";
import { Switch } from "@/ui/switch";
import { FieldRendererProps } from "./types";

export const BooleanFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  return (
    <Switch
      checked={(value as boolean) ?? field.defaultValue === "true" ?? false}
      onCheckedChange={onChange}
      className=""
    />
  );
};
