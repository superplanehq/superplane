import React from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";

export const DateFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  // Calculate min/max dates based on other fields for components like time gates
  const getDateConstraints = React.useMemo(() => {
    // For endDate field in time gates, prevent selecting dates before startDate
    if (field.name === "endDate" && allValues.startDate) {
      return { min: allValues.startDate as string };
    }

    // For startDate field in time gates, prevent selecting dates after endDate
    if (field.name === "startDate" && allValues.endDate) {
      return { max: allValues.endDate as string };
    }

    return {};
  }, [field.name, allValues.startDate, allValues.endDate]);

  return (
    <Input
      type="date"
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder="YYYY-MM-DD"
      className={hasError ? "border-red-500 border-2" : ""}
      min={getDateConstraints.min}
      max={getDateConstraints.max}
      data-testid={toTestId(`date-field-${field.name}`)}
    />
  );
};
