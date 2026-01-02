import { ConfigurationField } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArrowUpRight, PanelLeftClose, Plus, Settings, Trash2 } from "lucide-react";
import { useState } from "react";

export interface BlueprintMetadata {
  name: string;
  description: string;
}

export interface OutputChannel {
  name: string;
  nodeId: string;
  nodeOutputChannel: string;
}

export interface CustomComponentConfigurationSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;

  // Blueprint metadata
  metadata: BlueprintMetadata;
  onMetadataChange: (metadata: BlueprintMetadata) => void;

  // Configuration fields
  configurationFields: ConfigurationField[];
  onConfigurationFieldsChange: (fields: ConfigurationField[]) => void;
  onAddConfigField: () => void;
  onEditConfigField: (index: number) => void;

  // Output channels
  outputChannels: OutputChannel[];
  onOutputChannelsChange: (channels: OutputChannel[]) => void;
  onAddOutputChannel: () => void;
  onEditOutputChannel: (index: number) => void;
}

export function CustomComponentConfigurationSidebar({
  isOpen,
  onToggle,
  metadata,
  onMetadataChange,
  configurationFields,
  onConfigurationFieldsChange,
  onAddConfigField,
  onEditConfigField,
  outputChannels,
  onOutputChannelsChange,
  onAddOutputChannel,
  onEditOutputChannel,
}: CustomComponentConfigurationSidebarProps) {
  const [activeTab, setActiveTab] = useState<"configuration" | "outputChannels">("configuration");

  if (!isOpen) {
    return (
      <Button
        variant="outline"
        size="icon"
        onClick={() => onToggle(true)}
        aria-label="Open settings"
        className="absolute top-4 right-40 z-10"
        title="Bundle Settings"
      >
        <Settings size={24} />
      </Button>
    );
  }

  const handleDeleteConfigField = (index: number) => {
    const newFields = configurationFields.filter((_, i) => i !== index);
    onConfigurationFieldsChange(newFields);
  };

  const handleDeleteOutputChannel = (index: number) => {
    const newChannels = outputChannels.filter((_, i) => i !== index);
    onOutputChannelsChange(newChannels);
  };

  return (
    <div className="w-96 bg-white dark:bg-gray-900 border-l border-gray-200 dark:border-gray-800 flex flex-col z-50">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-0">
        <h2 className="text-md font-semibold text-gray-800 dark:text-gray-100">Bundle Settings</h2>
        <Button variant="outline" size="icon" onClick={() => onToggle(false)} aria-label="Close settings">
          <PanelLeftClose size={24} className="rotate-180" />
        </Button>
      </div>

      {/* Blueprint Metadata */}
      <div className="px-4 py-4 border-b border-gray-200 dark:border-gray-800">
        <div className="space-y-3">
          <div>
            <Label htmlFor="blueprint-name" className="text-xs font-medium text-gray-700 dark:text-gray-300">
              Name
            </Label>
            <Input
              id="blueprint-name"
              value={metadata.name}
              onChange={(e) => onMetadataChange({ ...metadata, name: e.target.value })}
              className="mt-1"
              placeholder="Blueprint name"
            />
          </div>
          <div>
            <Label htmlFor="blueprint-description" className="text-xs font-medium text-gray-700 dark:text-gray-300">
              Description
            </Label>
            <Input
              id="blueprint-description"
              value={metadata.description}
              onChange={(e) => onMetadataChange({ ...metadata, description: e.target.value })}
              className="mt-1"
              placeholder="Bundle description"
            />
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={(value: any) => setActiveTab(value)} className="flex-1 flex flex-col">
        <TabsList className="mx-4 mt-4 grid w-auto grid-cols-2">
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
          <TabsTrigger value="outputChannels">Output Channels</TabsTrigger>
        </TabsList>

        {/* Configuration Tab */}
        <TabsContent value="configuration" className="flex-1 overflow-y-auto mt-0">
          <div className="text-left p-4 space-y-6">
            <div className="!text-xs text-gray-500 dark:text-gray-400 mb-3">
              Add configuration fields that can be used in your component nodes
            </div>

            {/* Configuration Fields List */}
            {configurationFields.length > 0 && (
              <div className="space-y-4">
                {configurationFields.map((field: any, index: number) => (
                  <div
                    key={index}
                    className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3 cursor-pointer hover:border-blue-400 dark:hover:border-blue-600 transition-colors"
                    onClick={() => onEditConfigField(index)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <p className="font-medium text-sm text-gray-800 dark:text-gray-100">
                          {field.label || field.name}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          Type: {field.type} {field.required && "(required)"}
                        </p>
                        {field.description && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{field.description}</p>
                        )}
                        {field.type === "select" &&
                          field.typeOptions?.select?.options &&
                          field.typeOptions.select.options.length > 0 && (
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                              Options: {field.typeOptions.select.options.map((opt: any) => opt.label).join(", ")}
                            </p>
                          )}
                        {field.type === "multi_select" &&
                          field.typeOptions?.multiSelect?.options &&
                          field.typeOptions.multiSelect.options.length > 0 && (
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                              Options: {field.typeOptions.multiSelect.options.map((opt: any) => opt.label).join(", ")}
                            </p>
                          )}
                      </div>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteConfigField(index);
                        }}
                      >
                        <Trash2 className="text-red-500" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <Button variant="outline" onClick={onAddConfigField} className="w-full" data-testid="add-config-field-btn">
              <Plus />
              Add Configuration Field
            </Button>
          </div>
        </TabsContent>

        {/* Output Channels Tab */}
        <TabsContent value="outputChannels" className="flex-1 overflow-y-auto mt-0">
          <div className="text-left p-4 space-y-6">
            <div className="!text-xs text-gray-500 dark:text-gray-400 mb-3">
              Define output channels for this blueprint by selecting which node and channel should be exposed
            </div>

            {/* Output Channels List */}
            {outputChannels.length > 0 && (
              <div className="space-y-4">
                {outputChannels.map((outputChannel: any, index: number) => (
                  <div
                    key={index}
                    className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3 cursor-pointer hover:border-green-400 dark:hover:border-green-600 transition-colors"
                    onClick={() => onEditOutputChannel(index)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <ArrowUpRight className="text-green-600 dark:text-green-400" />
                          <p className="font-medium text-sm text-gray-800 dark:text-gray-100">{outputChannel.name}</p>
                        </div>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">Node: {outputChannel.nodeId}</p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          Channel: {outputChannel.nodeOutputChannel || "default"}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteOutputChannel(index);
                        }}
                      >
                        <Trash2 className="text-red-500" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Add New Output Channel Button */}
            <Button variant="outline" onClick={onAddOutputChannel} className="w-full">
              <Plus />
              Add Output Channel
            </Button>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
