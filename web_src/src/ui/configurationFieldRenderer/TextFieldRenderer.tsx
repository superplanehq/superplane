import React from "react";
import Editor from "@monaco-editor/react";
import type { FieldRendererProps } from "./types";
import { resolveIcon } from "@/lib/utils";
import { coerceMonacoValue } from "@/lib/monaco";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { toTestId } from "@/lib/testID";
import { useTheme } from "@/contexts/useTheme";
import { SimpleTooltip } from "../componentSidebar/SimpleTooltip";
import { useMonacoExpressionAutocomplete } from "./useMonacoExpressionAutocomplete";

const PLAIN_TEXT_MIN_HEIGHT_PX = 120;

const CODE_EDITOR_OPTIONS = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: false,
  autoIndent: "none" as const,
  formatOnPaste: false,
  formatOnType: false,
  tabSize: 2,
  insertSpaces: true,
  scrollBeyondLastLine: false,
  renderWhitespace: "boundary" as const,
  smoothScrolling: true,
  cursorBlinking: "smooth" as const,
  contextmenu: true,
  selectOnLineNumbers: true,
  bracketPairColorization: {
    enabled: true,
  },
  suggestOnTriggerCharacters: true,
  quickSuggestions: {
    other: true,
    strings: true,
    comments: false,
  },
  wordBasedSuggestions: "off" as const,
};

function resolveTextFieldLanguage(field: FieldRendererProps["field"]): string | undefined {
  const language = field.typeOptions?.text?.language?.trim();
  return language || undefined;
}

export const TextFieldRenderer: React.FC<FieldRendererProps> = (props) => {
  const language = resolveTextFieldLanguage(props.field);

  // Code fields (those that declare a language, e.g. scripts/commands) keep the
  // rich Monaco editor. Plain text fields (messages, descriptions, prompts) use a
  // lightweight multi-line editor instead.
  if (language) {
    return <CodeTextFieldRenderer {...props} language={language} />;
  }

  return <PlainTextFieldRenderer {...props} />;
};

const PlainTextFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  allowExpressions = false,
  excludedSuggestions,
  valuePreviewLabel,
}) => {
  const resolvedValue = value ?? field.defaultValue;
  const currentValue = resolvedValue == null ? "" : String(resolvedValue);
  const shouldPreserveEmpty = field.togglable === true;
  const emit = (nextValue: string) => onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined);

  if (!allowExpressions) {
    return (
      <Textarea
        value={currentValue}
        onChange={(e) => emit(e.target.value)}
        placeholder={field.placeholder || ""}
        style={{ minHeight: PLAIN_TEXT_MIN_HEIGHT_PX }}
        data-testid={toTestId(`text-field-${field.name}`)}
      />
    );
  }

  return (
    <AutoCompleteInput
      exampleObj={autocompleteExampleObj ?? null}
      value={currentValue}
      onChange={emit}
      placeholder={field.placeholder || ""}
      startWord="{{"
      prefix="{{ "
      suffix=" }}"
      inputSize="md"
      minHeight={PLAIN_TEXT_MIN_HEIGHT_PX}
      showValuePreview
      valuePreviewLabel={valuePreviewLabel}
      quickTip="Tip: type `{{` to start an expression."
      excludedSuggestions={excludedSuggestions}
      data-testid={toTestId(`text-field-${field.name}`)}
    />
  );
};

const CodeTextFieldRenderer: React.FC<FieldRendererProps & { language: string }> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  allowExpressions = false,
  language,
}) => {
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const { handleEditorMount } = useMonacoExpressionAutocomplete({
    autocompleteExampleObj: allowExpressions ? autocompleteExampleObj : null,
    languageId: language,
  });

  const editorValue = coerceMonacoValue(value ?? field.defaultValue);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editorValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleEditorChange = (newValue: string | undefined) => {
    const valueToUse = newValue || "";
    onChange(valueToUse || undefined);
  };

  return (
    <>
      <div className="flex flex-col gap-2 relative">
        <div className="border rounded-md border-gray-300 dark:border-gray-600 p-1" style={{ height: "200px" }}>
          <div className="absolute right-1.5 top-1.5 z-10 flex items-center gap-1">
            <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
              <button onClick={copyToClipboard} className="p-1 rounded text-gray-500 hover:text-gray-800">
                {React.createElement(resolveIcon("copy"), { size: 14 })}
              </button>
            </SimpleTooltip>
            <SimpleTooltip content="Expand">
              <button onClick={() => setIsModalOpen(true)} className="p-1 text-gray-500 hover:text-gray-800">
                {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
              </button>
            </SimpleTooltip>
          </div>
          <Editor
            height="100%"
            defaultLanguage={language}
            value={editorValue}
            onChange={handleEditorChange}
            onMount={allowExpressions ? handleEditorMount : undefined}
            theme={monacoTheme}
            options={CODE_EDITOR_OPTIONS}
          />
        </div>
      </div>

      {/* Expanded Editor Modal */}
      <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
        <DialogContent
          size="90vw"
          className="flex flex-col gap-0 overflow-hidden p-0"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex shrink-0 items-center justify-between border-b border-gray-200 px-4 py-3 pr-12 dark:border-gray-600">
            <DialogTitle>{field.label || field.name}</DialogTitle>
            <DialogDescription className="sr-only">
              Expanded text editor for {field.label || field.name}.
            </DialogDescription>
            <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  copyToClipboard();
                }}
                className="flex items-center gap-1 rounded bg-gray-50 px-3 py-1 text-sm text-gray-800 hover:bg-gray-200"
              >
                {React.createElement(resolveIcon("copy"), { size: 14 })}
                Copy
              </button>
            </SimpleTooltip>
          </div>
          <div className="min-h-0 flex-1 overflow-hidden">
            <Editor
              height="100%"
              defaultLanguage={language}
              value={editorValue}
              onChange={handleEditorChange}
              onMount={allowExpressions ? handleEditorMount : undefined}
              theme={monacoTheme}
              options={{
                ...CODE_EDITOR_OPTIONS,
                automaticLayout: true,
              }}
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};
