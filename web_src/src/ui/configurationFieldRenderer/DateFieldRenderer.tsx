import React from "react";
import { DatePickerField } from "./DatePickerField";
import { MonthDayFieldRenderer } from "./MonthDayFieldRenderer";
import { FieldRendererProps } from "./types";

export const DateFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  // Check if this is a recurring date field (no year) - used by timegate
  // Detect by checking if field name is "date" (used in timegate for specific day selection)
  const isRecurringDate = props.field.name === "date";

  if (isRecurringDate) {
    return <MonthDayFieldRenderer {...props} />;
  }

  return <DatePickerField {...props} />;
};
