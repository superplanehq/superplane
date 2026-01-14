import React, { useEffect } from "react";
import { Plus, X } from "lucide-react";
import { Button } from "../button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { showErrorToast } from "@/utils/toast";
import { TimeRangeWithAllDay } from "./TimeRangeWithAllDay";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface ExtendedFieldRendererProps extends FieldRendererProps {
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
  allValues?: Record<string, unknown>;
}

export const ListFieldRenderer: React.FC<ExtendedFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  hasError: _hasError,
  validationErrors,
  fieldPath = field.name || "",
  allValues: _allValues = {},
}) => {
  const items = Array.isArray(value) ? value : [];
  const listOptions = field.typeOptions?.list;
  const itemDefinition = listOptions?.itemDefinition;
  const itemLabel = listOptions?.itemLabel || "Item";
  const isApprovalItemsList =
    itemDefinition?.type === "object" &&
    Array.isArray(itemDefinition.schema) &&
    itemDefinition.schema.some((schemaField) => schemaField.name === "type") &&
    itemDefinition.schema.some((schemaField) => ["user", "role", "group"].includes(schemaField.name || ""));

  const getApproverKey = (item: Record<string, unknown>) => {
    const type = typeof item.type === "string" ? item.type : "";
    if (!type) return undefined;

    if (type === "user" && typeof item.user === "string" && item.user.trim()) {
      return `user:${item.user}`;
    }
    if (type === "role" && typeof item.role === "string" && item.role.trim()) {
      return `role:${item.role}`;
    }
    if (type === "group" && typeof item.group === "string" && item.group.trim()) {
      return `group:${item.group}`;
    }
    return undefined;
  };

  const addItem = () => {
    if (itemDefinition?.type === "object" && itemDefinition.schema) {
      // Initialize with default values from field definitions
      const newItem: Record<string, unknown> = {};

      // For exclude_dates, set default date only (no time fields - entire 24h excluded)
      if (field.name === "exclude_dates") {
        const dateField = itemDefinition.schema.find((f) => f.name === "date");

        if (dateField) {
          newItem.date = "12-31";
        }
      } else {
        // For timegate items (always weekly), set default days to weekdays and default times
        const daysField = itemDefinition.schema.find((f) => f.name === "days");
        const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
        const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");

        if (daysField) {
          // Set default days to weekdays (monday-friday)
          newItem.days = ["monday", "tuesday", "wednesday", "thursday", "friday"];
        }

        if (startTimeField && endTimeField) {
          // Set default times (00:00 to 23:59)
          newItem.startTime = "00:00";
          newItem.endTime = "23:59";
        }
      }

      onChange([...items, newItem]);
    } else {
      const newItem = itemDefinition?.type === "number" ? 0 : "";
      onChange([...items, newItem]);
    }
  };

  const removeItem = (index: number) => {
    const newItems = items.filter((_, i) => i !== index);
    onChange(newItems.length > 0 ? newItems : undefined);
  };

  const updateItem = (index: number, newValue: unknown) => {
    // For time windows, always ensure only one item exists
    if (isTimeWindows) {
      onChange([newValue]);
      return;
    }

    const newItems = [...items];
    newItems[index] = newValue;
    if (isApprovalItemsList) {
      const newKey =
        newValue && typeof newValue === "object" ? getApproverKey(newValue as Record<string, unknown>) : undefined;
      if (newKey) {
        const hasDuplicate = newItems.some((item, itemIndex) => {
          if (itemIndex === index || !item || typeof item !== "object") return false;
          return getApproverKey(item as Record<string, unknown>) === newKey;
        });
        if (hasDuplicate) {
          showErrorToast("Approver already added.");
          return;
        }
      }
    }
    onChange(newItems);
  };

  // Check if this is the items field (Time Windows) to add label
  const isTimeWindows = field.name === "items";

  // For time windows, ensure only one item exists
  useEffect(() => {
    if (isTimeWindows) {
      if (items.length === 0) {
        // Initialize with one default item if empty
        if (itemDefinition?.type === "object" && itemDefinition.schema) {
          const newItem: Record<string, unknown> = {};
          const daysField = itemDefinition.schema.find((f) => f.name === "days");
          const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
          const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");

          if (daysField) {
            newItem.days = ["monday", "tuesday", "wednesday", "thursday", "friday"];
          }

          if (startTimeField && endTimeField) {
            newItem.startTime = "00:00";
            newItem.endTime = "23:59";
          }

          onChange([newItem]);
        }
      } else if (items.length > 1) {
        // If multiple items exist, keep only the first one
        onChange([items[0]]);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isTimeWindows, items.length]);

  // For time windows, use only the first item
  const displayItems = isTimeWindows ? (items.length > 0 ? [items[0]] : []) : items;

  return (
    <div className={isTimeWindows ? "space-y-6" : "space-y-3"}>
      {displayItems.map((item, index) => {
        return (
          <div key={`item-${index}`} className={isTimeWindows ? "" : "relative"}>
            {itemDefinition?.type === "object" && itemDefinition.schema ? (
              <>
                {isTimeWindows ? (
                  // For time windows, render fields directly without wrapper
                  (() => {
                    const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
                    const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");
                    const daysField = itemDefinition.schema.find((f) => f.name === "days");

                    const itemValues =
                      item && typeof item === "object"
                        ? (item as Record<string, unknown>)
                        : ({} as Record<string, unknown>);
                    const nestedValues = isApprovalItemsList
                      ? {
                          ...itemValues,
                          __listItems: items,
                          __itemIndex: index,
                          __isApprovalList: true,
                        }
                      : itemValues;

                    const getNestedError = (fieldName: string) => {
                      const nestedFieldPath = `${fieldPath}[${index}].${fieldName}`;
                      if (!validationErrors) return false;
                      if (validationErrors instanceof Set) {
                        return validationErrors.has(nestedFieldPath);
                      } else {
                        return validationErrors.some((error) => error.field === nestedFieldPath);
                      }
                    };

                    // Check for time validation errors
                    const startTimeValue = itemValues.startTime as string | undefined;
                    const endTimeValue = itemValues.endTime as string | undefined;
                    const hasStartTimeError = getNestedError("startTime") || !startTimeValue || startTimeValue === "";
                    const hasEndTimeError = getNestedError("endTime") || !endTimeValue || endTimeValue === "";
                    const hasTimeError = hasStartTimeError || hasEndTimeError;

                    return (
                      <>
                        {/* Render days field for time windows - Group 1: Active Days */}
                        {daysField &&
                          (() => {
                            const daysValue = itemValues[daysField.name!] as string[] | undefined;
                            const daysArray = Array.isArray(daysValue) ? daysValue : [];
                            const hasDaysError = getNestedError(daysField.name!) || daysArray.length === 0;

                            return (
                              <div className="space-y-2 mb-6">
                                <Label className="block text-left">
                                  Active Days
                                  <span className="text-gray-800 dark:text-gray-300 ml-1">*</span>
                                </Label>
                                <ConfigurationFieldRenderer
                                  key={`days-${index}`}
                                  field={daysField}
                                  value={itemValues[daysField.name!]}
                                  onChange={(val) => {
                                    const newItem = { ...itemValues, [daysField.name!]: val };
                                    updateItem(index, newItem);
                                  }}
                                  allValues={nestedValues}
                                  domainId={domainId}
                                  domainType={domainType}
                                  hasError={hasDaysError}
                                />
                                {daysArray.length === 0 && (
                                  <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                    At least one day must be selected
                                  </p>
                                )}
                              </div>
                            );
                          })()}
                        {/* Render time range - Group 2: Active Time */}
                        {startTimeField && endTimeField && (
                          <div className="space-y-2">
                            <TimeRangeWithAllDay
                              key={`time-range-${index}`}
                              startTime={itemValues.startTime as string | undefined}
                              endTime={itemValues.endTime as string | undefined}
                              onStartTimeChange={(val) => {
                                const newItem = { ...itemValues, startTime: val };
                                updateItem(index, newItem);
                              }}
                              onEndTimeChange={(val) => {
                                const newItem = { ...itemValues, endTime: val };
                                updateItem(index, newItem);
                              }}
                              onBothTimesChange={(startVal, endVal) => {
                                // Update both times atomically
                                const newItem = { ...itemValues, startTime: startVal, endTime: endVal };
                                updateItem(index, newItem);
                              }}
                              hasError={hasTimeError}
                              hasStartTimeError={hasStartTimeError}
                              hasEndTimeError={hasEndTimeError}
                            />
                            {hasStartTimeError && (!startTimeValue || startTimeValue === "") && (
                              <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                Start time is required
                              </p>
                            )}
                            {hasEndTimeError && (!endTimeValue || endTimeValue === "") && (
                              <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                End time is required
                              </p>
                            )}
                          </div>
                        )}
                      </>
                    );
                  })()
                ) : (
                  // For non-time-windows, keep the wrapper with border and padding
                  <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
                    {(() => {
                      const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
                      const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");
                      const daysField = itemDefinition.schema.find((f) => f.name === "days");
                      const dateField = itemDefinition.schema.find((f) => f.name === "date");
                      const isExcludeDate = field.name === "exclude_dates";

                      const itemValues =
                        item && typeof item === "object"
                          ? (item as Record<string, unknown>)
                          : ({} as Record<string, unknown>);
                      const nestedValues = isApprovalItemsList
                        ? {
                            ...itemValues,
                            __listItems: items,
                            __itemIndex: index,
                            __isApprovalList: true,
                          }
                        : itemValues;

                      const getNestedError = (fieldName: string) => {
                        const nestedFieldPath = `${fieldPath}[${index}].${fieldName}`;
                        if (!validationErrors) return false;
                        if (validationErrors instanceof Set) {
                          return validationErrors.has(nestedFieldPath);
                        } else {
                          return validationErrors.some((error) => error.field === nestedFieldPath);
                        }
                      };

                      // Check for time validation errors
                      const startTimeValue = itemValues.startTime as string | undefined;
                      const endTimeValue = itemValues.endTime as string | undefined;
                      const hasStartTimeError = getNestedError("startTime") || !startTimeValue || startTimeValue === "";
                      const hasEndTimeError = getNestedError("endTime") || !endTimeValue || endTimeValue === "";
                      const hasTimeError = hasStartTimeError || hasEndTimeError;

                      return (
                        <>
                          {/* Render date field for exclude_dates */}
                          {isExcludeDate && dateField && (
                            <ConfigurationFieldRenderer
                              key={`date-${index}`}
                              field={dateField}
                              value={itemValues[dateField.name!]}
                              onChange={(val) => {
                                const newItem = { ...itemValues, [dateField.name!]: val };
                                updateItem(index, newItem);
                              }}
                              allValues={nestedValues}
                              domainId={domainId}
                              domainType={domainType}
                              hasError={getNestedError(dateField.name!)}
                            />
                          )}
                          {/* Render days field for non-time-windows (exclude_dates, etc.) */}
                          {daysField &&
                            (() => {
                              const daysValue = itemValues[daysField.name!] as string[] | undefined;
                              const daysArray = Array.isArray(daysValue) ? daysValue : [];
                              const hasDaysError = getNestedError(daysField.name!) || daysArray.length === 0;

                              return (
                                <div>
                                  <ConfigurationFieldRenderer
                                    key={`days-${index}`}
                                    field={daysField}
                                    value={itemValues[daysField.name!]}
                                    onChange={(val) => {
                                      const newItem = { ...itemValues, [daysField.name!]: val };
                                      updateItem(index, newItem);
                                    }}
                                    allValues={nestedValues}
                                    domainId={domainId}
                                    domainType={domainType}
                                    hasError={hasDaysError}
                                  />
                                  {daysArray.length === 0 && (
                                    <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                      At least one day must be selected
                                    </p>
                                  )}
                                </div>
                              );
                            })()}
                          {/* Render time range - Group 2: Active Time (skip for exclude_dates) */}
                          {startTimeField && endTimeField && !isExcludeDate && (
                            <div className="space-y-2">
                              <TimeRangeWithAllDay
                                key={`time-range-${index}`}
                                startTime={itemValues.startTime as string | undefined}
                                endTime={itemValues.endTime as string | undefined}
                                onStartTimeChange={(val) => {
                                  const newItem = { ...itemValues, startTime: val };
                                  updateItem(index, newItem);
                                }}
                                onEndTimeChange={(val) => {
                                  const newItem = { ...itemValues, endTime: val };
                                  updateItem(index, newItem);
                                }}
                                onBothTimesChange={(startVal, endVal) => {
                                  // Update both times atomically
                                  const newItem = { ...itemValues, startTime: startVal, endTime: endVal };
                                  updateItem(index, newItem);
                                }}
                                hasError={hasTimeError}
                                hasStartTimeError={hasStartTimeError}
                                hasEndTimeError={hasEndTimeError}
                              />
                              {hasStartTimeError && (!startTimeValue || startTimeValue === "") && (
                                <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                  Start time is required
                                </p>
                              )}
                              {hasEndTimeError && (!endTimeValue || endTimeValue === "") && (
                                <p className="text-xs text-red-500 dark:text-red-400 text-left mt-1">
                                  End time is required
                                </p>
                              )}
                            </div>
                          )}
                          {itemDefinition.schema
                            .filter(
                              (f) =>
                                f.name !== "startTime" &&
                                f.name !== "endTime" &&
                                f.name !== "days" &&
                                f.name !== "date",
                            )
                            .map((schemaField) => {
                              const itemValues =
                                item && typeof item === "object"
                                  ? (item as Record<string, unknown>)
                                  : ({} as Record<string, unknown>);
                              const nestedValues = isApprovalItemsList
                                ? {
                                    ...itemValues,
                                    __listItems: items,
                                    __itemIndex: index,
                                    __isApprovalList: true,
                                  }
                                : itemValues;

                              return (
                                <ConfigurationFieldRenderer
                                  key={schemaField.name}
                                  field={schemaField}
                                  value={itemValues[schemaField.name!]}
                                  onChange={(val) => {
                                    const newItem = { ...itemValues, [schemaField.name!]: val };
                                    updateItem(index, newItem);
                                  }}
                                  allValues={nestedValues}
                                  domainId={domainId}
                                  domainType={domainType}
                                  hasError={getNestedError(schemaField.name!)}
                                />
                              );
                            })}
                        </>
                      );
                    })()}
                  </div>
                )}
              </>
            ) : (
              <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4">
                <Input
                  type={itemDefinition?.type === "number" ? "number" : "text"}
                  value={item ?? ""}
                  onChange={(e) => {
                    const val =
                      itemDefinition?.type === "number"
                        ? e.target.value === ""
                          ? undefined
                          : Number(e.target.value)
                        : e.target.value;
                    updateItem(index, val);
                  }}
                />
              </div>
            )}
            {!isTimeWindows && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => removeItem(index)}
                      className="absolute top-2 right-2 h-6 w-6 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Remove item</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
        );
      })}
      {!isTimeWindows && (
        <Button variant="outline" onClick={addItem} className="w-full mt-3">
          <Plus className="h-4 w-4 mr-2" />
          Add {itemLabel}
        </Button>
      )}
    </div>
  );
};
