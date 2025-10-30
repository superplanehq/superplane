import { useState, useEffect } from "react";
import { Node } from "@xyflow/react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SuperplaneBlueprintsOutputChannel } from "@/api-client";

interface OutputChannelConfigurationModalProps {
  isOpen: boolean;
  onClose: () => void;
  outputChannel?: SuperplaneBlueprintsOutputChannel;
  nodes: Node[];
  onSave: (outputChannel: SuperplaneBlueprintsOutputChannel) => void;
}

export function OutputChannelConfigurationModal({
  isOpen,
  onClose,
  outputChannel,
  nodes,
  onSave,
}: OutputChannelConfigurationModalProps) {
  const [outputChannelForm, setOutputChannelForm] = useState<
    Partial<SuperplaneBlueprintsOutputChannel>
  >({
    name: "",
    nodeId: "",
    nodeOutputChannel: "default",
  });

  // Sync state when props change
  useEffect(() => {
    if (outputChannel) {
      setOutputChannelForm(outputChannel);
    } else {
      setOutputChannelForm({
        name: "",
        nodeId: "",
        nodeOutputChannel: "default",
      });
    }
  }, [outputChannel, isOpen]);

  const handleClose = () => {
    setOutputChannelForm({
      name: "",
      nodeId: "",
      nodeOutputChannel: "default",
    });
    onClose();
  };

  const handleSave = () => {
    if (!outputChannelForm.name?.trim() || !outputChannelForm.nodeId) {
      return;
    }

    onSave(outputChannelForm as SuperplaneBlueprintsOutputChannel);
    handleClose();
  };

  const selectedNode = nodes.find((n) => n.id === outputChannelForm.nodeId);
  const nodeChannels =
    (selectedNode?.data as any)?.outputChannels || ["default"];

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-2xl" showCloseButton={false}>
        <VisuallyHidden>
          <DialogTitle>
            {outputChannel ? "Edit Output Channel" : "Add Output Channel"}
          </DialogTitle>
          <DialogDescription>
            Configure the blueprint output channel
          </DialogDescription>
        </VisuallyHidden>
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-zinc-100 mb-6">
            {outputChannel ? "Edit Output Channel" : "Add Output Channel"}
          </h3>

          <div className="space-y-4">
            {/* Output Channel Name */}
            <div>
              <Label className="block text-sm font-medium mb-2">
                Output Channel Name *
              </Label>
              <Input
                type="text"
                value={outputChannelForm.name || ""}
                onChange={(e) =>
                  setOutputChannelForm({
                    ...outputChannelForm,
                    name: e.target.value,
                  })
                }
                placeholder="e.g., success, error, default"
                autoFocus
              />
              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                The name of this output channel
              </p>
            </div>

            {/* Node Selection */}
            <div>
              <Label className="block text-sm font-medium mb-2">Node *</Label>
              <Select
                value={outputChannelForm.nodeId || ""}
                onValueChange={(val) => {
                  // When node changes, reset the channel to default
                  setOutputChannelForm({
                    ...outputChannelForm,
                    nodeId: val,
                    nodeOutputChannel: "default",
                  });
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a node" />
                </SelectTrigger>
                <SelectContent>
                  {nodes
                    .filter((node) => node.type !== "outputChannel")
                    .map((node) => (
                      <SelectItem key={node.id} value={node.id}>
                        {(node.data as any).label} ({node.id})
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                Select which node's output to use for this channel
              </p>
            </div>

            {/* Node Output Channel Selection */}
            {outputChannelForm.nodeId && (
              <div>
                <Label className="block text-sm font-medium mb-2">
                  Node Output Channel *
                </Label>
                <Select
                  value={outputChannelForm.nodeOutputChannel || "default"}
                  onValueChange={(val) =>
                    setOutputChannelForm({
                      ...outputChannelForm,
                      nodeOutputChannel: val,
                    })
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {nodeChannels.map((channel: string) => (
                      <SelectItem key={channel} value={channel}>
                        {channel}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                  Select which output channel from the node to expose
                </p>
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
              disabled={
                !outputChannelForm.name?.trim() || !outputChannelForm.nodeId
              }
            >
              {outputChannel ? "Save Changes" : "Add Output Channel"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
