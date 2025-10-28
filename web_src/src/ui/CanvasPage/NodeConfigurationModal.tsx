import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { ComponentsConfigurationField } from "@/api-client";

interface NodeConfigurationModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  configuration: Record<string, any>;
  configurationFields: ComponentsConfigurationField[];
  onSave: (updatedConfiguration: Record<string, any>) => void;
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
  const [nodeConfiguration, setNodeConfiguration] = useState<Record<string, any>>(
    configuration || {}
  );

  const handleSave = () => {
    onSave(nodeConfiguration);
    onClose();
  };

  const handleClose = () => {
    // Reset to original configuration on cancel
    setNodeConfiguration(configuration || {});
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-2xl p-0" showCloseButton={false}>
        <ScrollArea className="max-h-[80vh]">
          <div className="p-6">
            <div className="space-y-6">
              {/* Node name section */}
              <div>
                <Label className="text-lg font-semibold">{nodeName}</Label>
                <p className="text-sm text-gray-500 dark:text-zinc-400 mt-1">
                  Edit node configuration
                </p>
              </div>

              {/* Configuration section */}
              {configurationFields && configurationFields.length > 0 ? (
                <div className="border-t border-gray-200 dark:border-zinc-700 pt-6 space-y-4">
                  {configurationFields.map((field) => (
                    <ConfigurationFieldRenderer
                      key={field.name}
                      field={field}
                      value={nodeConfiguration[field.name]}
                      onChange={(value) =>
                        setNodeConfiguration({
                          ...nodeConfiguration,
                          [field.name]: value,
                        })
                      }
                      allValues={nodeConfiguration}
                      domainId={domainId}
                      domainType={domainType}
                    />
                  ))}
                </div>
              ) : (
                <div className="border-t border-gray-200 dark:border-zinc-700 pt-6">
                  <p className="text-sm text-gray-500 dark:text-zinc-400">
                    No configuration fields available for this node.
                  </p>
                </div>
              )}
            </div>

            <DialogFooter className="mt-6">
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button variant="default" onClick={handleSave}>
                Save
              </Button>
            </DialogFooter>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
