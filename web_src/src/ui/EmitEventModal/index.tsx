import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import Editor from "@monaco-editor/react";
import { Play } from "lucide-react";
import type { editor } from "monaco-editor";
import { useEffect, useRef, useState } from "react";

interface EmitEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId: string;
  nodeName: string;
  workflowId: string;
  organizationId: string;
  channels: string[];
  onEmit: (channel: string, data: any) => Promise<void>;
  initialData?: string;
}

export const EmitEventModal = ({ isOpen, onClose, nodeName, channels, onEmit, initialData }: EmitEventModalProps) => {
  const [selectedChannel, setSelectedChannel] = useState<string>(channels[0] || "default");
  const [eventData, setEventData] = useState<string>(() => {
    if (initialData) {
      try {
        // Format the JSON with proper indentation
        return JSON.stringify(JSON.parse(initialData), null, 2);
      } catch {
        return initialData;
      }
    }
    return "{}";
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  // Update eventData when initialData changes
  useEffect(() => {
    if (initialData && isOpen) {
      try {
        setEventData(JSON.stringify(JSON.parse(initialData), null, 2));
      } catch {
        setEventData(initialData);
      }
    } else if (!initialData && isOpen) {
      setEventData("{}");
    }
  }, [initialData, isOpen]);

  // Cleanup editor when component unmounts
  useEffect(() => {
    return () => {
      if (editorRef.current) {
        editorRef.current.dispose();
        editorRef.current = null;
      }
    };
  }, []);

  const handleClose = () => {
    setSelectedChannel(channels[0] || "default");
    if (initialData) {
      try {
        setEventData(JSON.stringify(JSON.parse(initialData), null, 2));
      } catch {
        setEventData(initialData);
      }
    } else {
      setEventData("{}");
    }
    setIsSubmitting(false);
    onClose();
  };

  const handleSubmit = async () => {
    try {
      setIsSubmitting(true);

      // Validate JSON
      let parsedData;
      try {
        parsedData = JSON.parse(eventData);
      } catch (e) {
        showErrorToast("Invalid JSON format");
        setIsSubmitting(false);
        return;
      }

      await onEmit(selectedChannel, parsedData);
      showSuccessToast("Event emitted successfully");
      handleClose();
    } catch (error) {
      console.error("Failed to emit event", error);
      showErrorToast("Failed to emit event");
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="max-w-3xl max-h-[80vh]">
        <div className="space-y-4">
          <DialogTitle className="flex items-center gap-3">
            <Play className="text-blue-600 dark:text-blue-400" size={24} />
            Emit Event
          </DialogTitle>

          <DialogDescription className="text-sm text-gray-500 dark:text-gray-400">
            Manually emit an output event for node: <strong>{nodeName}</strong>
          </DialogDescription>

          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <Label className="min-w-[120px]">Output Channel</Label>
              <Select value={selectedChannel} onValueChange={setSelectedChannel}>
                <SelectTrigger className="flex-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {channels.map((channel) => (
                    <SelectItem key={channel} value={channel}>
                      {channel}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="border border-gray-200 dark:border-gray-700 rounded-md overflow-hidden">
              <Editor
                height="300px"
                defaultLanguage="json"
                value={eventData}
                onChange={(value) => setEventData(value || "{}")}
                onMount={(editor) => {
                  editorRef.current = editor;
                }}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: "on",
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                }}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              data-testid="emit-event-cancel-button"
              variant="outline"
              onClick={handleClose}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button data-testid="emit-event-submit-button" onClick={handleSubmit} disabled={isSubmitting}>
              <Play size={16} />
              {isSubmitting ? "Emitting..." : "Emit Event"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
};
