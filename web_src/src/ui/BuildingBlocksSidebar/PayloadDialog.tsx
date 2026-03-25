import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import Editor from "@monaco-editor/react";
import { Copy, Check } from "lucide-react";
import { useState } from "react";

interface PayloadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  label: string;
  payloadString: string;
}

export function PayloadDialog({ open, onOpenChange, title, label, payloadString }: PayloadDialogProps) {
  const [copied, setCopied] = useState(false);
  const lineCount = payloadString.split("\n").length;
  const lineHeight = 19;
  const editorHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 500);

  const handleCopy = () => {
    navigator.clipboard.writeText(payloadString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="w-[60vw] max-w-[60vw] h-auto max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <div>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription className="text-sm text-gray-500">{label}</DialogDescription>
          </div>
          <button
            onClick={handleCopy}
            className="px-3 py-1 text-sm text-gray-800 bg-gray-50 hover:bg-gray-200 rounded flex items-center gap-1"
          >
            {copied ? <Check size={14} /> : <Copy size={14} />}
            {copied ? "Copied" : "Copy"}
          </button>
        </div>
        <div className="flex-1 overflow-hidden border border-gray-200 dark:border-gray-700 rounded-md">
          <Editor
            height={`${editorHeight}px`}
            defaultLanguage="json"
            value={payloadString}
            theme="vs"
            options={{
              readOnly: true,
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              wordWrap: "on",
              folding: true,
              scrollBeyondLastLine: false,
              renderWhitespace: "none",
              contextmenu: true,
              domReadOnly: true,
              scrollbar: {
                vertical: "auto",
                horizontal: "auto",
              },
            }}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
