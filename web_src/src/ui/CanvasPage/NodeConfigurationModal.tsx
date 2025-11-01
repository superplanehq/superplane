import { useState, useEffect, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { ComponentsConfigurationField } from "@/api-client";

interface NodeConfigurationModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  configuration: Record<string, unknown>;
  configurationFields: ComponentsConfigurationField[];
  onSave: (updatedConfiguration: Record<string, unknown>, updatedNodeName: string) => void;
  domainId?: string;
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION";
}

export function NodeConfigurationModal({
  isOpen,
  onClose,
  nodeName,
  configuration,
  configurationFields,
  onSave,
  domainId,
  domainType,
}: NodeConfigurationModalProps) {
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, unknown>>(
    configuration || {}
  );
  const [currentNodeName, setCurrentNodeName] = useState<string>(nodeName);
  const [validationErrors, setValidationErrors] = useState<Set<string>>(new Set());
  const [showValidation, setShowValidation] = useState(false);

  const isFieldEmpty = (value: unknown): boolean => {
    if (value === null || value === undefined) return true;
    if (typeof value === 'string') return value.trim() === '';
    if (Array.isArray(value)) return value.length === 0;
    if (typeof value === 'object') return Object.keys(value).length === 0;
    return false;
  };

  const validateAllFields = useCallback((): boolean => {
    const errors = new Set<string>();

    configurationFields.forEach((field) => {
      if (field.required && field.name) {
        const value = nodeConfiguration[field.name];
        if (isFieldEmpty(value)) {
          errors.add(field.name);
        }
      }
    });

    setValidationErrors(errors);
    setShowValidation(true);
    return errors.size === 0;
  }, [configurationFields, nodeConfiguration]);

  // Sync state when props change (e.g., when modal opens for a different node)
  useEffect(() => {
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    setValidationErrors(new Set());
    setShowValidation(false);
  }, [configuration, nodeName]);

  const handleSave = () => {
    if (validateAllFields()) {
      onSave(nodeConfiguration, currentNodeName);
      onClose();
    }
  };

  const handleClose = () => {
    // Reset to original configuration and name on cancel
    setNodeConfiguration(configuration || {});
    setCurrentNodeName(nodeName);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-2xl p-0" showCloseButton={false}>
        <ScrollArea className="max-h-[80vh]">
          <div className="p-6">
            <div className="space-y-6">
              {/* Node identification section */}
              <div className="flex items-center gap-3">
                <Label className="min-w-[100px] text-left">Node Name</Label>
                <Input
                  type="text"
                  value={currentNodeName}
                  onChange={(e) => setCurrentNodeName(e.target.value)}
                  placeholder="Enter a name for this node"
                  autoFocus
                  className="flex-1"
                />
              </div>

              {/* Configuration section */}
              {configurationFields && configurationFields.length > 0 && (
                <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
                  {configurationFields.map((field) => {
                    if (!field.name) return null;
                    const fieldName = field.name;
                    return (
                      <ConfigurationFieldRenderer
                        key={fieldName}
                        field={field}
                        value={nodeConfiguration[fieldName]}
                        onChange={(value) =>
                          setNodeConfiguration({
                            ...nodeConfiguration,
                            [fieldName]: value,
                          })
                        }
                        allValues={nodeConfiguration}
                        domainId={domainId}
                        domainType={domainType}
                        hasError={showValidation && validationErrors.has(fieldName)}
                      />
                    );
                  })}
                </div>
              )}
            </div>

            <DialogFooter className="mt-6">
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={handleSave}
              >
                Add
              </Button>
            </DialogFooter>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
