import Placeholder from "@tiptap/extension-placeholder";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

import { pasteMarkdownFromClipboard } from "./lib/workOrderDescriptionMarkdown";
import { WorkspaceUnderline } from "./lib/workspaceUnderline";
import { WorkOrderDescriptionFormatToolbar } from "./WorkOrderDescriptionFormatToolbar";

interface WorkOrderDescriptionEditorProps {
  value: string;
  maxLength: number;
  disabled: boolean;
  onChange: (next: string) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  className?: string;
}

export function WorkOrderDescriptionEditor({
  value,
  maxLength,
  disabled,
  onChange,
  onFocus,
  onBlur,
  className,
}: WorkOrderDescriptionEditorProps) {
  const editorRef = useRef<Editor | null>(null);
  const onChangeRef = useRef(onChange);
  const onFocusRef = useRef(onFocus);
  const onBlurRef = useRef(onBlur);
  const maxLengthRef = useRef(maxLength);
  const emittedMarkdownRef = useRef(value);
  const pasteInFlightRef = useRef(false);

  onChangeRef.current = onChange;
  onFocusRef.current = onFocus;
  onBlurRef.current = onBlur;
  maxLengthRef.current = maxLength;

  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3, 4] },
        link: { openOnClick: false, autolink: true },
        underline: false,
      }),
      WorkspaceUnderline,
      Markdown,
      Placeholder.configure({ placeholder: "Add description…" }),
    ],
    content: value,
    contentType: "markdown",
    editable: !disabled,
    editorProps: {
      attributes: {
        id: "work-order-description-input",
        class: cn(
          "work-order-description-editor workspace-markdown min-h-40 outline-none",
          className ?? "text-[15px] leading-6",
        ),
        "data-testid": "work-order-description-input",
      },
      handlePaste: (_view, event) => {
        const text = event.clipboardData?.getData("text/plain") ?? "";
        if (!text) {
          return false;
        }
        pasteInFlightRef.current = true;
        queueMicrotask(() => {
          pasteInFlightRef.current = false;
        });
        const current = editorRef.current;
        if (!current) {
          pasteInFlightRef.current = false;
          return false;
        }
        return pasteMarkdownFromClipboard(current, event);
      },
    },
    onUpdate: ({ editor: current }) => {
      const markdown = current.getMarkdown();
      const limit = maxLengthRef.current;
      const previous = emittedMarkdownRef.current;
      const fromPaste = pasteInFlightRef.current;
      pasteInFlightRef.current = false;
      const next = markdownWithinLimit(markdown, previous, limit, fromPaste);
      if (next !== markdown) {
        current.commands.setContent(next, {
          contentType: "markdown",
          emitUpdate: false,
        });
      }
      if (next === previous) {
        return;
      }
      emittedMarkdownRef.current = next;
      onChangeRef.current(next);
    },
    onFocus: () => {
      onFocusRef.current?.();
    },
    onBlur: () => {
      onBlurRef.current?.();
    },
  });

  editorRef.current = editor;

  useEffect(() => {
    if (!editor) {
      return;
    }
    editor.setEditable(!disabled);
  }, [disabled, editor]);

  useEffect(() => {
    if (!editor) {
      return;
    }
    if (value === emittedMarkdownRef.current) {
      return;
    }
    emittedMarkdownRef.current = value;
    editor.commands.setContent(value, { contentType: "markdown" });
  }, [editor, value]);

  return (
    <>
      {editor ? (
        <BubbleMenu
          editor={editor}
          updateDelay={0}
          appendTo={() =>
            editor.view.dom.closest('[data-testid="create-work-order-dialog"]') ??
            editor.view.dom.closest('[data-testid="work-order-split-run"]') ??
            editor.view.dom.parentElement ??
            document.body
          }
          options={{ placement: "top", offset: 8, strategy: "fixed" }}
        >
          <WorkOrderDescriptionFormatToolbar disabled={disabled} editor={editor} />
        </BubbleMenu>
      ) : null}
      <EditorContent editor={editor} />
    </>
  );
}

function markdownWithinLimit(markdown: string, previous: string, limit: number, fromPaste: boolean): string {
  if (markdown.length <= limit) {
    return markdown;
  }
  if (fromPaste) {
    return markdown.slice(0, limit);
  }
  return previous;
}
