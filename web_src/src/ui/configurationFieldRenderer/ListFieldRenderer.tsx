import React from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../button";
import { Input } from "@/components/ui/input";
import { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { showErrorToast } from "@/utils/toast";
import { TimeRangeWithAllDay } from "./TimeRangeWithAllDay";

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
  allValues = {},
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
      
      // Find the type field and set its default value (first option for select fields)
      const typeField = itemDefinition.schema.find((f) => f.name === "type");
      if (typeField && typeField.type === "select" && typeField.typeOptions?.select?.options && typeField.typeOptions.select.options.length > 0) {
        // Use the first option as default (which should be "weekly")
        const defaultType = typeField.typeOptions.select.options[0].value;
        newItem.type = defaultType;
        
        // For weekly type, don't preselect any days (all turned off by default)
        if (defaultType === "weekly") {
          newItem.days = [];
        }
        // For specific_dates type, set default date to December 31 (12-31)
        else if (defaultType === "specific_dates") {
          newItem.date = "12-31";
        }
      }
      
      // For timegate items, set "All day" to ON by default (00:00 to 23:59)
      const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
      const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");
      if (startTimeField && endTimeField) {
        newItem.startTime = "00:00";
        newItem.endTime = "23:59";
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

  // Check if this is timegate items field and determine label based on when_to_run
  const isTimegateItems = field.name === "items";
  const whenToRun = allValues?.when_to_run as string | undefined;
  return (
    <div className="space-y-3">
      {items.map((item, index) => {
        const itemType = (item && typeof item === "object" ? (item as Record<string, unknown>).type : undefined) as string | undefined;
        return (
        <div key={`item-${index}-${itemType || 'new'}`} className="flex gap-2 items-center">
          <div className="flex-1">
            {itemDefinition?.type === "object" && itemDefinition.schema ? (
              <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
                {(() => {
                  // Check if we have a type field (for timegate items)
                  const typeField = itemDefinition.schema.find((f) => f.name === "type");
                  const startTimeField = itemDefinition.schema.find((f) => f.name === "startTime");
                  const endTimeField = itemDefinition.schema.find((f) => f.name === "endTime");
                  const dateField = itemDefinition.schema.find((f) => f.name === "date");
                  const daysField = itemDefinition.schema.find((f) => f.name === "days");
                  
                  // For timegate, render type field first, then handle time fields specially, then other fields
                  if (typeField) {
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

                    const hasTimeError = getNestedError("startTime") || getNestedError("endTime");
                    // Get itemType from itemValues, or fallback to default (first option) if not set
                    let itemType = itemValues.type as string | undefined;
                    if (!itemType && typeField.type === "select" && typeField.typeOptions?.select?.options && typeField.typeOptions.select.options.length > 0) {
                      itemType = typeField.typeOptions.select.options[0].value as string;
                    }
                    const isSpecificDate = itemType === "specific_dates";
                    const isWeekly = itemType === "weekly";

                    return (
                      <>
                        <ConfigurationFieldRenderer
                          key={`type-${index}-${itemType}`}
                          field={typeField}
                          value={itemValues[typeField.name!]}
                          onChange={(val) => {
                            const newItem = { ...itemValues, [typeField.name!]: val };
                            // When switching to "specific_dates", set default date to Dec 31 if not already set
                            if (val === "specific_dates" && !newItem.date) {
                              newItem.date = "12-31";
                            }
                            // When switching to "weekly", don't set default days (all turned off by default)
                            if (val === "weekly" && !newItem.days) {
                              newItem.days = [];
                            }
                            updateItem(index, newItem);
                          }}
                          allValues={nestedValues}
                          domainId={domainId}
                          domainType={domainType}
                          hasError={getNestedError(typeField.name!)}
                        />
                        {/* Render days field for weekly */}
                        {isWeekly && daysField && (
                          <ConfigurationFieldRenderer
                            key={`days-${index}-${itemType}`}
                            field={daysField}
                            value={itemValues[daysField.name!]}
                            onChange={(val) => {
                              const newItem = { ...itemValues, [daysField.name!]: val };
                              updateItem(index, newItem);
                            }}
                            allValues={nestedValues}
                            domainId={domainId}
                            domainType={domainType}
                            hasError={getNestedError(daysField.name!)}
                          />
                        )}
                        {/* Render date field for specific dates */}
                        {isSpecificDate && dateField && (
                          <ConfigurationFieldRenderer
                            key={`date-${index}-${itemType}`}
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
                        {/* Render time range with "All day" toggle if both time fields exist */}
                        {startTimeField && endTimeField && (
                          <TimeRangeWithAllDay
                            key={`time-range-${index}-${itemType}`}
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
                            itemType={itemType}
                          />
                        )}
                        {itemDefinition.schema
                          .filter((f) => f.name !== "type" && f.name !== "startTime" && f.name !== "endTime" && f.name !== "date" && f.name !== "days")
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
                  }

                  // Default rendering for other fields
                  return itemDefinition.schema.map((schemaField) => {
                    const nestedFieldPath = `${fieldPath}[${index}].${schemaField.name}`;
                    const hasNestedError = (() => {
                      if (!validationErrors) return false;
                      if (validationErrors instanceof Set) {
                        return validationErrors.has(nestedFieldPath);
                      } else {
                        return validationErrors.some((error) => error.field === nestedFieldPath);
                      }
                    })();

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
                        hasError={hasNestedError}
                      />
                    );
                  });
                })()}
              </div>
            ) : (
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
            )}
          </div>
          <Button variant="ghost" size="icon" onClick={() => removeItem(index)} className="mt-1">
            <Trash2 className="h-4 w-4 text-red-500" />
          </Button>
        </div>
        );
      })}
      <Button variant="outline" onClick={addItem} className="w-full mt-3">
        <Plus className="h-4 w-4 mr-2" />
        Add {itemLabel}
      </Button>
    </div>
  );
};
