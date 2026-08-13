import Placeholder from "@tiptap/extension-placeholder";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import StarterKit from "@tiptap/starter-kit";
import { useEffect, useRef } from "react";

import { pasteMarkdownFromClipboard } from "./lib/workOrderDescriptionMarkdown";
import { WorkOrderDescriptionFormatToolbar } from "./WorkOrderDescriptionFormatToolbar";

interface WorkOrderDescriptionEditorProps {
  value: string;
  maxLength: number;
  disabled: boolean;
  onChange: (next: string) => void;
  onFocus?: () => void;
  onBlur?: () => void;
}

export function WorkOrderDescriptionEditor({
  value,
  maxLength,
  disabled,
  onChange,
  onFocus,
  onBlur,
}: WorkOrderDescriptionEditorProps) {
  const editorRef = useRef<Editor | null>(null);
  const onChangeRef = useRef(onChange);
  const onFocusRef = useRef(onFocus);
  const onBlurRef = useRef(onBlur);
  const maxLengthRef = useRef(maxLength);
  const emittedMarkdownRef = useRef(value);

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
      }),
      Markdown,
      Placeholder.configure({ placeholder: "Add description…" }),
    ],
    content: value,
    contentType: "markdown",
    editable: !disabled,
    editorProps: {
      attributes: {
        id: "work-order-description-input",
        class: "work-order-description-editor workspace-markdown min-h-40 text-[15px] leading-6 outline-none",
        "data-testid": "work-order-description-input",
      },
      handlePaste: (_view, event) => {
        const current = editorRef.current;
        if (!current) {
          return false;
        }
        return pasteMarkdownFromClipboard(current, event);
      },
    },
    onUpdate: ({ editor: current }) => {
      const markdown = current.getMarkdown();
      if (markdown.length > maxLengthRef.current) {
        current.commands.undo();
        return;
      }
      emittedMarkdownRef.current = markdown;
      onChangeRef.current(markdown);
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
