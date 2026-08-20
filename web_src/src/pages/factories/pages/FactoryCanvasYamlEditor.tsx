import { Editor } from "@monaco-editor/react";

import { useTheme } from "@/contexts/useTheme";

type FactoryCanvasYamlEditorProps = {
  value: string;
  readOnly?: boolean;
  onChange?: (value: string) => void;
};

export function FactoryCanvasYamlEditor({ value, readOnly = true, onChange }: FactoryCanvasYamlEditorProps) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  return (
    <Editor
      height="100%"
      language="yaml"
      value={value}
      theme={monacoTheme}
      onChange={readOnly ? undefined : (next) => onChange?.(next ?? "")}
      options={{
        readOnly,
        domReadOnly: readOnly,
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: "on",
        wordWrap: "on",
        folding: true,
        scrollBeyondLastLine: false,
        renderWhitespace: "boundary",
        smoothScrolling: true,
        tabSize: 2,
        renderLineHighlight: "line",
      }}
    />
  );
}
