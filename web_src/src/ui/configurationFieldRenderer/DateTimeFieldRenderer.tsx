import React from "react";
import { DateTimePickerField } from "./DateTimePickerField";
import { FieldRendererProps } from "./types";

export const DateTimeFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  return <DateTimePickerField {...props} />;
};
