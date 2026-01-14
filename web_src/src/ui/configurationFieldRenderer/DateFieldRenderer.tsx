import React from "react";
import { MonthDayFieldRenderer } from "./MonthDayFieldRenderer";
import { FieldRendererProps } from "./types";

export const DateFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  // DateFieldRenderer is only used for recurring date fields (no year) - used by timegate
  // The field name is "date" (used in timegate exclude_dates for specific day selection)
  return <MonthDayFieldRenderer {...props} />;
};
