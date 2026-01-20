import React from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";

export const DateTimeFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  // Calculate min/max datetime based on other fields for components like time gates
  const getDateTimeConstraints = React.useMemo(() => {
    // For endDateTime field in time gates, prevent selecting datetimes before startDateTime
    if (field.name === "endDateTime" && allValues.startDateTime) {
      return { min: allValues.startDateTime as string };
    }

    // For startDateTime field in time gates, prevent selecting datetimes after endDateTime
    if (field.name === "startDateTime" && allValues.endDateTime) {
      return { max: allValues.endDateTime as string };
    }

    return {};
  }, [field.name, allValues.startDateTime, allValues.endDateTime]);

  return (
    <Input
      type="datetime-local"
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder="YYYY-MM-DDTHH:MM"
      className=""
      min={getDateTimeConstraints.min}
      max={getDateTimeConstraints.max}
    />
  );
};
