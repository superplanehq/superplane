import React from "react";
import Editor from "@monaco-editor/react";
import type { FieldRendererProps } from "./types";
import { resolveIcon } from "@/lib/utils";
import { coerceMonacoValue } from "@/lib/monaco";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
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
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const resolvedValue = value ?? field.defaultValue;
  const currentValue = resolvedValue == null ? "" : String(resolvedValue);
  const shouldPreserveEmpty = field.togglable === true;
  const emit = (nextValue: string) => onChange(shouldPreserveEmpty ? nextValue : nextValue || undefined);
  const testId = `text-field-${field.name}`;
  const label = field.label || field.name || "value";

  const inlineEditor = allowExpressions ? (
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
      data-testid={toTestId(testId)}
    />
  ) : (
    <Textarea
      value={currentValue}
      onChange={(e) => emit(e.target.value)}
      placeholder={field.placeholder || ""}
      style={{ minHeight: PLAIN_TEXT_MIN_HEIGHT_PX }}
      data-testid={toTestId(testId)}
    />
  );

  return (
    <div className="relative">
      {inlineEditor}
      <ExpandFieldButton
        onClick={() => setIsModalOpen(true)}
        label={`Expand ${label} editor`}
        testId={toTestId(`${testId}-expand`)}
      />
      <ExpandableEditorDialog
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        title={label}
        initialValue={currentValue}
        onSave={emit}
        testId={toTestId(`${testId}-modal`)}
      >
        {({ value: draftValue, onChange: setDraftValue }) =>
          allowExpressions ? (
            <AutoCompleteInput
              exampleObj={autocompleteExampleObj ?? null}
              value={draftValue}
              onChange={setDraftValue}
              placeholder={field.placeholder || ""}
              startWord="{{"
              prefix="{{ "
              suffix=" }}"
              inputSize="md"
              showValuePreview
              valuePreviewLabel={valuePreviewLabel}
              quickTip="Tip: type `{{` to start an expression."
              excludedSuggestions={excludedSuggestions}
              fullHeight
              data-testid={toTestId(`${testId}-modal-input`)}
            />
          ) : (
            <Textarea
              value={draftValue}
              onChange={(e) => setDraftValue(e.target.value)}
              placeholder={field.placeholder || ""}
              // `field-sizing-fixed` overrides the default `field-sizing-content` from the shadcn
              // Textarea primitive so the control fills the dialog instead of tracking content height.
              className="h-full min-h-0 flex-1 resize-none field-sizing-fixed"
              data-testid={toTestId(`${testId}-modal-input`)}
            />
          )
        }
      </ExpandableEditorDialog>
    </div>
  );
};

const CodeTextFieldRenderer: React.FC<FieldRendererProps & { language: string }> = ({
  field,
  value,
  onChange,
  autocompleteExampleObj,
  language,
}) => {
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const { handleEditorMount } = useMonacoExpressionAutocomplete({
    autocompleteExampleObj,
    languageId: language,
  });

  const editorValue = coerceMonacoValue(value ?? field.defaultValue);
  const testId = `text-field-${field.name}`;
  const label = field.label || field.name || "value";

  const copyToClipboard = (source: string) => {
    navigator.clipboard.writeText(source);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleEditorChange = (newValue: string | undefined) => {
    const valueToUse = newValue || "";
    onChange(valueToUse || undefined);
  };

  const commitDraft = (draft: string) => {
    onChange(draft || undefined);
  };

  return (
    <>
      <div className="flex flex-col gap-2 relative">
        <div className="border rounded-md border-gray-300 dark:border-gray-600 p-1" style={{ height: "200px" }}>
          <div className="absolute right-1.5 top-1.5 z-10 flex items-center gap-1">
            <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
              <button
                type="button"
                aria-label={`Copy ${label}`}
                onClick={() => copyToClipboard(editorValue)}
                className="p-1 rounded text-gray-500 hover:text-gray-800"
              >
                {React.createElement(resolveIcon("copy"), { size: 14 })}
              </button>
            </SimpleTooltip>
            <SimpleTooltip content="Expand">
              <button
                type="button"
                aria-label={`Expand ${label} editor`}
                data-testid={toTestId(`${testId}-expand`)}
                onClick={() => setIsModalOpen(true)}
                className="p-1 text-gray-500 hover:text-gray-800"
              >
                {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
              </button>
            </SimpleTooltip>
          </div>
          <Editor
            height="100%"
            defaultLanguage={language}
            value={editorValue}
            onChange={handleEditorChange}
            onMount={handleEditorMount}
            theme={monacoTheme}
            options={CODE_EDITOR_OPTIONS}
          />
        </div>
      </div>

      <ExpandableEditorDialog
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        title={label}
        initialValue={editorValue}
        onSave={commitDraft}
        testId={toTestId(`${testId}-modal`)}
        headerActions={({ draft }) => (
          <SimpleTooltip content={copied ? "Copied!" : "Copy"} hideOnClick={false}>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                copyToClipboard(draft);
              }}
              className="flex items-center gap-1 rounded bg-gray-50 px-3 py-1 text-sm text-gray-800 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-100 dark:hover:bg-gray-700"
            >
              {React.createElement(resolveIcon("copy"), { size: 14 })}
              Copy
            </button>
          </SimpleTooltip>
        )}
      >
        {({ value: draftValue, onChange: setDraftValue }) => (
          <Editor
            height="100%"
            defaultLanguage={language}
            value={draftValue}
            onChange={(next) => setDraftValue(next ?? "")}
            onMount={handleEditorMount}
            theme={monacoTheme}
            options={{
              ...CODE_EDITOR_OPTIONS,
              automaticLayout: true,
            }}
          />
        )}
      </ExpandableEditorDialog>
    </>
  );
};

interface ExpandableEditorDialogChildProps {
  value: string;
  onChange: (next: string) => void;
}

interface ExpandableEditorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  initialValue: string;
  onSave: (next: string) => void;
  testId?: string;
  children: (props: ExpandableEditorDialogChildProps) => React.ReactNode;
  headerActions?: (props: { draft: string }) => React.ReactNode;
}

const ExpandableEditorDialog: React.FC<ExpandableEditorDialogProps> = ({
  open,
  onOpenChange,
  title,
  initialValue,
  onSave,
  testId,
  children,
  headerActions,
}) => {
  const [draft, setDraft] = React.useState(initialValue);
  const wasOpenRef = React.useRef(open);

  // Reset the draft only when the dialog transitions from closed to open so that
  // Cancel/Escape/overlay-click reliably discards in-progress edits, and so that
  // an unrelated parent-side value update while editing does not overwrite the draft.
  React.useEffect(() => {
    if (open && !wasOpenRef.current) {
      setDraft(initialValue);
    }
    wasOpenRef.current = open;
  }, [open, initialValue]);

  const handleSave = () => {
    onSave(draft);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  // Autocomplete suggestions render in a portal on `document.body`, which Radix Dialog treats
  // as an outside interaction (and would otherwise dismiss the dialog while the user is picking
  // a suggestion). Prevent the default close when the interaction originates in that portal.
  const handleInteractOutside = (event: Event) => {
    const target = event.target as Element | null;
    if (target?.closest?.("[data-autocomplete-suggestions]")) {
      event.preventDefault();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="90vw"
        className="flex flex-col gap-0 overflow-hidden p-0"
        onClick={(e) => e.stopPropagation()}
        onPointerDownOutside={handleInteractOutside}
        onFocusOutside={handleInteractOutside}
        onInteractOutside={handleInteractOutside}
        data-testid={testId}
      >
        <div className="flex shrink-0 items-center justify-between gap-2 border-b border-gray-200 px-4 py-3 pr-12 dark:border-gray-600">
          <DialogTitle className="truncate">{title}</DialogTitle>
          <DialogDescription className="sr-only">
            Expanded editor for {title}. Save to apply your changes or cancel to discard them.
          </DialogDescription>
          {headerActions ? <div className="flex items-center gap-2">{headerActions({ draft })}</div> : null}
        </div>
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden px-4 py-3">
          {children({ value: draft, onChange: setDraft })}
        </div>
        <DialogFooter className="shrink-0 border-t border-gray-200 px-4 py-3 dark:border-gray-600">
          <Button type="button" variant="outline" onClick={handleCancel} data-testid="expandable-editor-cancel">
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} data-testid="expandable-editor-save">
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

interface ExpandFieldButtonProps {
  onClick: () => void;
  label: string;
  testId?: string;
}

const ExpandFieldButton: React.FC<ExpandFieldButtonProps> = ({ onClick, label, testId }) => (
  <SimpleTooltip content="Expand">
    <button
      type="button"
      aria-label={label}
      data-testid={testId}
      onClick={onClick}
      className="absolute right-1.5 top-1.5 z-10 rounded bg-white/80 p-1 text-gray-500 backdrop-blur-sm hover:text-gray-800 dark:bg-gray-800/80 dark:text-gray-400 dark:hover:text-gray-100"
    >
      {React.createElement(resolveIcon("maximize-2"), { size: 14 })}
    </button>
  </SimpleTooltip>
);
