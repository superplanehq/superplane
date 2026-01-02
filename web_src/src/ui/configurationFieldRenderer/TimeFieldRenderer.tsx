import React, { useEffect } from "react";
import { Input } from "../input";
import { FieldRendererProps } from "./types";

export const TimeFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  useEffect(() => {
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
      }
    }
  }, [value, field.defaultValue, onChange]);

  // Calculate min/max based on other fields for time gates
  const getTimeConstraints = React.useMemo(() => {
    // For endTime field in time gates, prevent selecting times before startTime
    if (field.name === "endTime" && allValues.startTime) {
      return { min: allValues.startTime as string };
    }

    // For startTime field in time gates, prevent selecting times after endTime
    if (field.name === "startTime" && allValues.endTime) {
      return { max: allValues.endTime as string };
    }

    return {};
  }, [field.name, allValues.startTime, allValues.endTime]);

  return (
    <Input
      type="time"
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.typeOptions?.time?.format || "HH:MM"}
      className={hasError ? "border-red-500 border-2" : ""}
      min={getTimeConstraints.min}
      max={getTimeConstraints.max}
      data-testid={`time-field-${field.name}`}
    />
  );
};
