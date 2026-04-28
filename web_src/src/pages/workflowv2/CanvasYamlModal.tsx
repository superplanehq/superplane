import { Copy, Download, Upload, Text, WrapText } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

import { Editor } from "@monaco-editor/react";
import { ImportYamlIntoCanvasDialog } from "./ImportYamlIntoCanvasDialog";

export type CanvasYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;

  yamlText: string;
  filename: string;
  onCopy?: () => void;
  onDownload?: () => void;
  onImport?: (data: { nodes: unknown[]; edges: unknown[] }) => Promise<void>;
  isImporting?: boolean;
};

export function CanvasYamlModal(props: CanvasYamlModalProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] h-full flex-col gap-0 overflow-hidden p-0">
        <DialogTitle className="sr-only">Canvas YAML</DialogTitle>

        <div className="flex h-full flex-col">
          <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2">
            <span className="font-mono text-sm text-gray-600">{props.filename}</span>
            <div className="flex items-center gap-2 mr-8">
              <ImportButton {...props} />
              <CopyButton {...props} />
              <DownloadButton {...props} />
            </div>
          </div>

          <YamlEditor {...props} />
        </div>
      </DialogContent>
    </Dialog>
  );
}

function YamlEditor(props: CanvasYamlModalProps) {
  const [wordWrap, setWordWrap] = useState(true);

  return (
    <div className="canvas-yaml-monaco h-full min-h-0 min-w-0">
      <div className="flex items-center justify-between px-4 py-1 border-b border-gray-200">
        <span className="text-xs text-gray-500">YAML Preview</span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setWordWrap(!wordWrap)}
          title={wordWrap ? "Disable word wrap" : "Enable word wrap"}
        >
          {wordWrap ? <WrapText className="h-4 w-4" /> : <Text className="h-4 w-4" />}
        </Button>
      </div>
      <Editor
        height="calc(100% - 40px)"
        language="yaml"
        value={props.yamlText}
        theme="vs"
        options={{
          readOnly: true,
          domReadOnly: true,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: "on",
          wordWrap: wordWrap ? "on" : "off",
          folding: true,
          scrollBeyondLastLine: false,
          renderWhitespace: "boundary",
          smoothScrolling: true,
          tabSize: 2,
          renderLineHighlight: "line",
          renderLineHighlightOnlyWhenFocus: false,
        }}
      />
    </div>
  );
}

function ImportButton(props: CanvasYamlModalProps) {
  if (!props.onImport) {
    return null;
  }

  return <ImportButtonWithDialog onImport={props.onImport} isImporting={props.isImporting} />;
}

type ImportButtonWithDialogProps = {
  onImport: NonNullable<CanvasYamlModalProps["onImport"]>;
  isImporting?: boolean;
};

function ImportButtonWithDialog({ onImport, isImporting }: ImportButtonWithDialogProps) {
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false);

  return (
    <>
      <Button variant="outline" size="sm" onClick={() => setIsImportDialogOpen(true)}>
        <Upload />
        Import
      </Button>
      <ImportYamlIntoCanvasDialog
        open={isImportDialogOpen}
        onOpenChange={setIsImportDialogOpen}
        onImport={onImport}
        isImporting={isImporting}
      />
    </>
  );
}

function CopyButton(props: CanvasYamlModalProps) {
  if (!props.onCopy) return null;

  return (
    <Button variant="outline" size="sm" onClick={() => navigator.clipboard.writeText(props.yamlText)}>
      <Copy />
      Copy
    </Button>
  );
}

function DownloadButton(props: CanvasYamlModalProps) {
  if (!props.onDownload) return null;

  return (
    <Button variant="outline" size="sm" onClick={props.onDownload}>
      <Download />
      Download
    </Button>
  );
}
