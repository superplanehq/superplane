import React from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../button";
import { Switch } from "@/ui/switch";
import { FieldRendererProps, ValidationError } from "./types";
import { ConfigurationFieldRenderer } from "./index";

interface ExtendedFieldRendererProps extends FieldRendererProps {
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}

export const TogglableListFieldRenderer: React.FC<ExtendedFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  hasError,
  validationErrors,
  fieldPath = field.name || "",
}) => {
  // Determine if the field is enabled (has a non-null value)
  const isEnabled = value !== null && value !== undefined;

  // Get the array value (empty array if disabled)
  const items = isEnabled && Array.isArray(value) ? value : [];

  // Get options from field type options
  // Note: Using any cast until API types are regenerated to include togglableList
  const listOptions = (field.typeOptions as any)?.togglableList;
  const itemDefinition = listOptions?.itemDefinition;
  const itemLabel = listOptions?.itemLabel || "Item";

  const handleToggleChange = (checked: boolean) => {
    if (checked) {
      // Enable the field with empty array
      onChange([]);
    } else {
      // Disable the field by setting to null
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
    onChange(newItems.length > 0 ? newItems : []);
  };

  const updateItem = (index: number, newValue: unknown) => {
    if (!isEnabled) return;

    const newItems = [...items];
    newItems[index] = newValue;
    onChange(newItems);
  };

  return (
    <div className="flex items-start gap-3">
      <Switch
        checked={isEnabled}
        onCheckedChange={handleToggleChange}
        className={`mt-2 ${hasError ? "border-red-500 border-2" : ""}`}
      />
      <div className={`flex-1 space-y-3 ${!isEnabled ? "opacity-50" : ""}`}>
        {isEnabled &&
          items.map((item, index) => (
            <div key={index} className="flex gap-2 items-center">
              <div className="flex-1">
                {itemDefinition?.type === "object" && itemDefinition.schema ? (
                  <div className="border border-gray-300 dark:border-gray-700 rounded-md p-4 space-y-4">
                    {itemDefinition.schema.map((schemaField: any) => (
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
};
