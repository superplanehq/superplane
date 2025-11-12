import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogFooter, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, Trash2 } from "lucide-react";
import { ConfigurationField, ConfigurationSelectOption } from "@/api-client";

interface ConfigurationFieldModalProps {
  isOpen: boolean;
  onClose: () => void;
  field?: ConfigurationField;
  onSave: (field: ConfigurationField) => void;
}

export function ConfigurationFieldModal({ isOpen, onClose, field, onSave }: ConfigurationFieldModalProps) {
  const [configFieldForm, setConfigFieldForm] = useState<Partial<ConfigurationField>>({
    name: "",
    label: "",
    type: "string",
    description: "",
    placeholder: "",
    required: false,
    typeOptions: {},
  });

  // Sync state when props change
  useEffect(() => {
    if (field) {
      setConfigFieldForm(field);
    } else {
      setConfigFieldForm({
        name: "",
        label: "",
        type: "string",
        description: "",
        placeholder: "",
        required: false,
        typeOptions: {},
      });
    }
  }, [field, isOpen]);

  const handleClose = () => {
    setConfigFieldForm({
      name: "",
      label: "",
      type: "string",
      description: "",
      placeholder: "",
      required: false,
      typeOptions: {},
    });
    onClose();
  };

  const handleSave = () => {
    if (!configFieldForm.name?.trim()) {
      return;
    }

    // Validate options for select/multi_select types
    if (configFieldForm.type === "select") {
      const options = configFieldForm.typeOptions?.select?.options || [];
      if (options.length === 0) {
        return;
      }

      // Validate that all options have both label and value
      const hasInvalidOption = options.some((opt) => !opt.label?.trim() || !opt.value?.trim());
      if (hasInvalidOption) {
        return;
      }
    } else if (configFieldForm.type === "multi_select") {
      const options = configFieldForm.typeOptions?.multiSelect?.options || [];
      if (options.length === 0) {
        return;
      }

      // Validate that all options have both label and value
      const hasInvalidOption = options.some((opt) => !opt.label?.trim() || !opt.value?.trim());
      if (hasInvalidOption) {
        return;
      }
    }

    onSave(configFieldForm as ConfigurationField);
    handleClose();
  };

  const isSelect = configFieldForm.type === "select";
  const currentOptions = isSelect
    ? configFieldForm.typeOptions?.select?.options || []
    : configFieldForm.typeOptions?.multiSelect?.options || [];

  const updateOptions = (newOptions: ConfigurationSelectOption[]) => {
    if (isSelect) {
      setConfigFieldForm({
        ...configFieldForm,
        typeOptions: {
          ...configFieldForm.typeOptions,
          select: { options: newOptions },
        },
      });
    } else {
      setConfigFieldForm({
        ...configFieldForm,
        typeOptions: {
          ...configFieldForm.typeOptions,
          multiSelect: { options: newOptions },
        },
      });
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-2xl" showCloseButton={false}>
        <VisuallyHidden>
          <DialogTitle>{field ? "Edit Configuration Field" : "Add Configuration Field"}</DialogTitle>
          <DialogDescription>Configure the blueprint configuration field</DialogDescription>
        </VisuallyHidden>
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-zinc-100 mb-6">
            {field ? "Edit Configuration Field" : "Add Configuration Field"}
          </h3>

          <div className="space-y-4">
            {/* Field Name */}
            <div>
              <Label className="block text-sm font-medium mb-2">Field Name *</Label>
              <Input
                type="text"
                value={configFieldForm.name || ""}
                onChange={(e) => setConfigFieldForm({ ...configFieldForm, name: e.target.value })}
                placeholder="e.g., threshold_expression"
                autoFocus
              />
              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                This is the internal name used in templates (e.g., $config.threshold_expression)
              </p>
            </div>

            {/* Field Label */}
            <div>
              <Label className="block text-sm font-medium mb-2">Label *</Label>
              <Input
                type="text"
                value={configFieldForm.label || ""}
                onChange={(e) =>
                  setConfigFieldForm({
                    ...configFieldForm,
                    label: e.target.value,
                  })
                }
                placeholder="e.g., Threshold Expression"
              />
              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">Display name shown in the UI</p>
            </div>

            {/* Field Type */}
            <div>
              <Label className="block text-sm font-medium mb-2">Type *</Label>
              <Select
                value={configFieldForm.type || "string"}
                onValueChange={(val) => setConfigFieldForm({ ...configFieldForm, type: val })}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="string">String</SelectItem>
                  <SelectItem value="number">Number</SelectItem>
                  <SelectItem value="boolean">Boolean</SelectItem>
                  <SelectItem value="select">Select</SelectItem>
                  <SelectItem value="multi_select">Multi-Select</SelectItem>
                  <SelectItem value="date">Date</SelectItem>
                  <SelectItem value="url">URL</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Options Section (for select and multi_select types) */}
            {(configFieldForm.type === "select" || configFieldForm.type === "multi_select") && (
              <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <Label className="block text-sm font-medium">Options *</Label>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      updateOptions([...currentOptions, { label: "", value: "" }]);
                    }}
                  >
                    <Plus />
                    Add Option
                  </Button>
                </div>

                {currentOptions.length > 0 ? (
                  <div className="space-y-2">
                    {currentOptions.map((option, index: number) => (
                      <div key={index} className="flex gap-2 items-start">
                        <div className="flex-1 grid grid-cols-2 gap-2">
                          <Input
                            type="text"
                            value={option.label || ""}
                            onChange={(e) => {
                              const newOptions = [...currentOptions];
                              newOptions[index] = {
                                ...option,
                                label: e.target.value,
                              };
                              updateOptions(newOptions);
                            }}
                            placeholder="Label (e.g., Low)"
                          />
                          <Input
                            type="text"
                            value={option.value || ""}
                            onChange={(e) => {
                              const newOptions = [...currentOptions];
                              newOptions[index] = {
                                ...option,
                                value: e.target.value,
                              };
                              updateOptions(newOptions);
                            }}
                            placeholder="Value (e.g., low)"
                          />
                        </div>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => {
                            const newOptions = currentOptions.filter((_, i: number) => i !== index);
                            updateOptions(newOptions);
                          }}
                        >
                          <Trash2 className="text-red-500" />
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-gray-500 dark:text-zinc-400">
                    No options added yet. Click "Add Option" to add options.
                  </p>
                )}
              </div>
            )}

            {/* Field Description */}
            <div>
              <Label className="block text-sm font-medium mb-2">Description</Label>
              <Input
                type="text"
                value={configFieldForm.description || ""}
                onChange={(e) =>
                  setConfigFieldForm({
                    ...configFieldForm,
                    description: e.target.value,
                  })
                }
                placeholder="Describe the purpose of this field"
              />
            </div>

            {/* Field Placeholder */}
            <div>
              <Label className="block text-sm font-medium mb-2">Placeholder</Label>
              <Input
                type="text"
                value={configFieldForm.placeholder || ""}
                onChange={(e) =>
                  setConfigFieldForm({
                    ...configFieldForm,
                    placeholder: e.target.value,
                  })
                }
                placeholder="Optional placeholder text"
              />
            </div>

            {/* Required Checkbox */}
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={configFieldForm.required || false}
                onChange={(e) =>
                  setConfigFieldForm({
                    ...configFieldForm,
                    required: e.target.checked,
                  })
                }
                className="h-4 w-4 rounded border-gray-300 dark:border-zinc-700"
                id="required-checkbox"
              />
              <Label htmlFor="required-checkbox" className="cursor-pointer">
                Required field
              </Label>
            </div>
          </div>

          <DialogFooter className="mt-6">
            <Button variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              variant="default"
              onClick={handleSave}
              disabled={!configFieldForm.name?.trim() || !configFieldForm.label?.trim()}
            >
              {field ? "Save Changes" : "Add Field"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
