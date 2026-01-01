import React from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../button";
import { Input } from "../input";
import { Switch } from "@/ui/switch";
import { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";

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
  hasError: _hasError,
  validationErrors,
  fieldPath = field.name || "",
}) => {
  const isTogglable = field.togglable === true;

  const isEnabled = isTogglable ? value !== null && value !== undefined : true;

  const items = isEnabled && Array.isArray(value) ? value : [];
  const listOptions = field.typeOptions?.list;
  const itemDefinition = listOptions?.itemDefinition;
  const itemLabel = listOptions?.itemLabel || "Item";

  const handleToggleChange = (checked: boolean) => {
    if (!isTogglable) return;

    if (checked) {
      onChange([]);
    } else {
      onChange(null);
    }
  };

  const addItem = () => {
    if (!isEnabled) return;

    const newItem = itemDefinition?.type === "object" ? {} : itemDefinition?.type === "number" ? 0 : "";
    onChange([...items, newItem]);
  };

  const removeItem = (index: number) => {
    if (!isEnabled) return;

    const newItems = items.filter((_, i) => i !== index);
    onChange(newItems.length > 0 ? newItems : isTogglable ? [] : undefined);
  };

  const updateItem = (index: number, newValue: unknown) => {
    if (!isEnabled) return;

    const newItems = [...items];
    newItems[index] = newValue;
    onChange(newItems);
  };

  if (isTogglable) {
    return (
      <div className="flex items-start gap-3">
        <Switch
          checked={isEnabled}
          onCheckedChange={handleToggleChange}
          className={`mt-2 ${_hasError ? "border-red-500 border-2" : ""}`}
        />
        <div className={`flex-1 space-y-3 ${!isEnabled ? "opacity-50" : ""}`}>
          {isEnabled &&
            items.map((item, index) => (
              <div key={index} className="flex gap-2 items-center">
                <div className="flex-1">
                  {itemDefinition?.type === "object" && itemDefinition.schema ? (
                    <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
                      {itemDefinition.schema.map((schemaField) => (
                        <ConfigurationFieldRenderer
                          key={`${schemaField.name}-${index}`}
                          field={schemaField}
                          value={item?.[schemaField.name!] ?? undefined}
                          onChange={(newValue) => {
                            const updatedItem = { ...item, [schemaField.name!]: newValue };
                            updateItem(index, updatedItem);
                          }}
                          domainId={domainId}
                          domainType={domainType}
                          validationErrors={validationErrors}
                          fieldPath={`${fieldPath}[${index}].${schemaField.name}`}
                        />
                      ))}
                    </div>
                  ) : (
                    <ConfigurationFieldRenderer
                      field={{
                        name: `${field.name}_item_${index}`,
                        label: "",
                        type: itemDefinition?.type || "string",
                        required: false,
                      }}
                      value={item}
                      onChange={(newValue) => updateItem(index, newValue)}
                      domainId={domainId}
                      domainType={domainType}
                    />
                  )}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => removeItem(index)}
                  disabled={!isEnabled}
                  className="flex-shrink-0"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}

          {isEnabled && (
            <Button type="button" variant="outline" size="sm" onClick={addItem} className="w-full">
              <Plus className="h-4 w-4 mr-2" />
              Add {itemLabel}
            </Button>
          )}

          {!isEnabled && (
            <div className="text-sm text-gray-500 dark:text-gray-400 italic">
              Toggle on to add {itemLabel.toLowerCase()}s
            </div>
          )}
        </div>
      </div>
    );
  }

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

                  return (
                    <ConfigurationFieldRenderer
                      key={schemaField.name}
                      field={schemaField}
                      value={item[schemaField.name!]}
                      onChange={(val) => {
                        const newItem = { ...item, [schemaField.name!]: val };
                        updateItem(index, newItem);
                      }}
                      allValues={item}
                      domainId={domainId}
                      domainType={domainType}
                      hasError={hasNestedError}
                    />
                  );
                })}
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
      ))}
      <Button variant="outline" onClick={addItem} className="w-full mt-3">
        <Plus className="h-4 w-4 mr-2" />
        Add {itemLabel}
      </Button>
    </div>
  );
};
