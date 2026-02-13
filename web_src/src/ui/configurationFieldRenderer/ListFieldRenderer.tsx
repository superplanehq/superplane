import React from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../button";
import { Input } from "@/components/ui/input";
import { DayInYearFieldRenderer } from "./DayInYearFieldRenderer";
import { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";
import { showErrorToast } from "@/utils/toast";

interface ExtendedFieldRendererProps extends FieldRendererProps {
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}

export const ListFieldRenderer: React.FC<ExtendedFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  appInstallationId,
  organizationId,
  hasError: _,
  validationErrors,
  fieldPath = field.name || "",
  autocompleteExampleObj,
  allowExpressions = false,
}) => {
  const listOptions = field.typeOptions?.list;
  const itemDefinition = listOptions?.itemDefinition;
  const maxItems = listOptions?.maxItems;
  const items = Array.isArray(value)
    ? itemDefinition?.type === "day-in-year"
      ? value.filter((item) => typeof item === "string" && item.trim().length > 0)
      : value
    : [];
  const itemLabel = listOptions?.itemLabel || "Item";
  const canAddMore = maxItems === undefined || items.length < maxItems;
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
    const newItem =
      itemDefinition?.type === "object"
        ? {}
        : itemDefinition?.type === "number"
          ? 0
          : itemDefinition?.type === "day-in-year"
            ? "01/01"
            : "";
    onChange([...items, newItem]);
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

  return (
    <div className="space-y-3">
      {items.map((item, index) => (
        <div key={index} className="flex gap-2 items-center">
          <div className="flex-1">
            {itemDefinition?.type === "object" && itemDefinition.schema ? (
              <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
                {itemDefinition.schema.map((schemaField) => {
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
                      allowExpressions={allowExpressions}
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
                      appInstallationId={appInstallationId}
                      organizationId={organizationId}
                      hasError={hasNestedError}
                      autocompleteExampleObj={autocompleteExampleObj}
                    />
                  );
                })}
              </div>
            ) : itemDefinition?.type === "day-in-year" ? (
              <DayInYearFieldRenderer
                field={{ name: `${field.name || "item"}-${index}`, label: itemLabel, type: "day-in-year" }}
                value={item}
                onChange={(val) => updateItem(index, val)}
              />
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
      ))}
      <Button variant="outline" onClick={addItem} className="w-full mt-3" disabled={!canAddMore}>
        <Plus className="h-4 w-4 mr-2" />
        Add {itemLabel}
      </Button>
    </div>
  );
};
