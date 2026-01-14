import React from "react";
import { DatePickerField } from "./DatePickerField";
import { FieldRendererProps } from "./types";

export const DateFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  return <DatePickerField {...props} />;
};
