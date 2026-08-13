import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useEditorState, type Editor } from "@tiptap/react";
import {
  Bold,
  ChevronDown,
  Code,
  Italic,
  Link as LinkIcon,
  List,
  Quote,
  SquareCode,
  Strikethrough,
  Underline,
} from "lucide-react";
import { useState, type ReactNode } from "react";

import { hrefFromInput } from "./lib/workOrderDescriptionHref";

type HeadingLevel = 1 | 2 | 3 | 4;

const HEADING_OPTIONS: Array<{ level: HeadingLevel | 0; label: string }> = [
  { level: 0, label: "Paragraph" },
  { level: 1, label: "Heading 1" },
  { level: 2, label: "Heading 2" },
  { level: 3, label: "Heading 3" },
  { level: 4, label: "Heading 4" },
];

interface WorkOrderDescriptionFormatToolbarProps {
  editor: Editor;
  disabled: boolean;
}

export function WorkOrderDescriptionFormatToolbar({ editor, disabled }: WorkOrderDescriptionFormatToolbarProps) {
  const [headingMenuOpen, setHeadingMenuOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [linkHref, setLinkHref] = useState("");
  const [portalRoot, setPortalRoot] = useState<HTMLDivElement | null>(null);
  const format = useEditorState({
    editor,
    selector: ({ editor: current }) => ({
      headingLevel: activeHeadingLevel(current),
      bold: current.isActive("bold"),
      italic: current.isActive("italic"),
      underline: current.isActive("underline"),
      strike: current.isActive("strike"),
      code: current.isActive("code"),
      codeBlock: current.isActive("codeBlock"),
      blockquote: current.isActive("blockquote"),
      bulletList: current.isActive("bulletList"),
      link: current.isActive("link"),
    }),
  });

  const applyHeading = (level: HeadingLevel | 0) => {
    if (level === 0) {
      editor.chain().focus().setParagraph().run();
    } else {
      editor.chain().focus().toggleHeading({ level }).run();
    }
    setHeadingMenuOpen(false);
  };

  const applyLink = () => {
    const href = hrefFromInput(linkHref);
    if (!href) {
      editor.chain().focus().unsetLink().run();
    } else {
      editor.chain().focus().setLink({ href }).run();
    }
    setLinkOpen(false);
  };

  return (
    <div
      ref={setPortalRoot}
      className="z-50 flex flex-col rounded-md border border-border bg-popover text-popover-foreground shadow-md"
      data-testid="work-order-description-toolbar"
    >
      <FormatToolbarActions
        editor={editor}
        disabled={disabled}
        format={format}
        headingMenuOpen={headingMenuOpen}
        headingMenuContainer={portalRoot}
        linkOpen={linkOpen}
        onOpenChangeHeadingMenu={(nextOpen) => {
          setHeadingMenuOpen(nextOpen);
          if (nextOpen) {
            setLinkOpen(false);
          }
        }}
        onSelectHeading={applyHeading}
        onToggleLink={() => {
          setHeadingMenuOpen(false);
          setLinkHref(editor.getAttributes("link").href ?? "");
          setLinkOpen((current) => !current);
        }}
      />
      {linkOpen ? <LinkField href={linkHref} onHrefChange={setLinkHref} onApply={applyLink} /> : null}
    </div>
  );
}

function FormatToolbarActions({
  editor,
  disabled,
  format,
  headingMenuOpen,
  headingMenuContainer,
  linkOpen,
  onOpenChangeHeadingMenu,
  onSelectHeading,
  onToggleLink,
}: {
  editor: Editor;
  disabled: boolean;
  format: {
    headingLevel: HeadingLevel | 0;
    bold: boolean;
    italic: boolean;
    underline: boolean;
    strike: boolean;
    code: boolean;
    codeBlock: boolean;
    blockquote: boolean;
    bulletList: boolean;
    link: boolean;
  };
  headingMenuOpen: boolean;
  headingMenuContainer: HTMLElement | null;
  linkOpen: boolean;
  onOpenChangeHeadingMenu: (open: boolean) => void;
  onSelectHeading: (level: HeadingLevel | 0) => void;
  onToggleLink: () => void;
}) {
  return (
    <div className="flex items-center gap-0.5 p-0.5">
      <HeadingMenu
        disabled={disabled}
        headingLevel={format.headingLevel}
        open={headingMenuOpen}
        container={headingMenuContainer}
        onOpenChange={onOpenChangeHeadingMenu}
        onSelect={onSelectHeading}
      />
      <ToolbarDivider />
      <FormatButton
        label="Bold"
        disabled={disabled}
        pressed={format.bold}
        onMouseDown={() => editor.chain().focus().toggleBold().run()}
      >
        <Bold className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton
        label="Italic"
        disabled={disabled}
        pressed={format.italic}
        onMouseDown={() => editor.chain().focus().toggleItalic().run()}
      >
        <Italic className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton
        label="Underline"
        disabled={disabled}
        pressed={format.underline}
        onMouseDown={() => editor.chain().focus().toggleUnderline().run()}
      >
        <Underline className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton
        label="Strikethrough"
        disabled={disabled}
        pressed={format.strike}
        onMouseDown={() => editor.chain().focus().toggleStrike().run()}
      >
        <Strikethrough className="size-3.5" aria-hidden />
      </FormatButton>
      <ToolbarDivider />
      <FormatButton
        label="Code"
        disabled={disabled}
        pressed={format.code}
        onMouseDown={() => editor.chain().focus().toggleCode().run()}
      >
        <Code className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton
        label="Code block"
        disabled={disabled}
        pressed={format.codeBlock}
        onMouseDown={() => editor.chain().focus().toggleCodeBlock().run()}
      >
        <SquareCode className="size-3.5" aria-hidden />
      </FormatButton>
      <ToolbarDivider />
      <FormatButton
        label="Quote"
        disabled={disabled}
        pressed={format.blockquote}
        onMouseDown={() => editor.chain().focus().toggleBlockquote().run()}
      >
        <Quote className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton
        label="List"
        disabled={disabled}
        pressed={format.bulletList}
        onMouseDown={() => editor.chain().focus().toggleBulletList().run()}
      >
        <List className="size-3.5" aria-hidden />
      </FormatButton>
      <ToolbarDivider />
      <FormatButton label="Link" disabled={disabled} pressed={format.link || linkOpen} onMouseDown={onToggleLink}>
        <LinkIcon className="size-3.5" aria-hidden />
      </FormatButton>
    </div>
  );
}

function LinkField({
  href,
  onHrefChange,
  onApply,
}: {
  href: string;
  onHrefChange: (next: string) => void;
  onApply: () => void;
}) {
  return (
    <form
      className="flex items-center gap-1 border-t border-border px-1.5 py-1"
      onSubmit={(event) => {
        event.preventDefault();
        onApply();
      }}
    >
      <Input
        aria-label="Link URL"
        value={href}
        onChange={(event) => onHrefChange(event.target.value)}
        placeholder="https://"
        className="h-7 min-w-44 text-[12px]"
      />
      <Button type="submit" size="sm" className="h-7 px-2 text-[12px]">
        Apply
      </Button>
    </form>
  );
}

function HeadingMenu({
  disabled,
  headingLevel,
  open,
  container,
  onOpenChange,
  onSelect,
}: {
  disabled: boolean;
  headingLevel: HeadingLevel | 0;
  open: boolean;
  container: HTMLElement | null;
  onOpenChange: (open: boolean) => void;
  onSelect: (level: HeadingLevel | 0) => void;
}) {
  return (
    <DropdownMenu modal={false} open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          disabled={disabled}
          aria-label="Heading"
          className="h-7 gap-0.5 rounded-sm px-1.5 text-[12px] font-medium text-muted-foreground"
          onMouseDown={(event) => {
            event.preventDefault();
          }}
        >
          {headingLevel === 0 ? "P" : `H${headingLevel}`}
          <ChevronDown className="size-3" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        container={container}
        className="min-w-36"
        onOpenAutoFocus={(event) => event.preventDefault()}
        onCloseAutoFocus={(event) => event.preventDefault()}
      >
        {HEADING_OPTIONS.map((option) => (
          <DropdownMenuItem
            key={option.label}
            className={cn(
              "text-[12px]",
              option.level === headingLevel ? "font-medium text-foreground" : "text-muted-foreground",
            )}
            onMouseDown={(event) => event.preventDefault()}
            onSelect={() => onSelect(option.level)}
          >
            {option.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function FormatButton({
  label,
  disabled,
  pressed,
  onMouseDown,
  children,
}: {
  label: string;
  disabled: boolean;
  pressed?: boolean;
  onMouseDown: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={label}
      aria-pressed={pressed}
      disabled={disabled}
      className={cn("size-7 text-muted-foreground", pressed && "bg-accent text-foreground")}
      onMouseDown={(event) => {
        event.preventDefault();
        onMouseDown();
      }}
    >
      {children}
    </Button>
  );
}

function ToolbarDivider() {
  return <span aria-hidden className="mx-0.5 h-4 w-px bg-border" />;
}

function activeHeadingLevel(editor: Editor): HeadingLevel | 0 {
  for (const level of [1, 2, 3, 4] as const) {
    if (editor.isActive("heading", { level })) {
      return level;
    }
  }
  return 0;
}
