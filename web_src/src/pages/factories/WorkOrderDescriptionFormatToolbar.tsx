import { Button } from "@/components/ui/button";
import { Bold, Code, Italic, List } from "lucide-react";
import type { ReactNode, RefObject } from "react";

interface WorkOrderDescriptionFormatToolbarProps {
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  value: string;
  maxLength: number;
  disabled: boolean;
  onChange: (next: string) => void;
}

export function WorkOrderDescriptionFormatToolbar({
  textareaRef,
  value,
  maxLength,
  disabled,
  onChange,
}: WorkOrderDescriptionFormatToolbarProps) {
  const applyWrap = (prefix: string, suffix = prefix) => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const selected = value.slice(start, end) || "text";
    const next = `${value.slice(0, start)}${prefix}${selected}${suffix}${value.slice(end)}`;
    if (next.length > maxLength) {
      return;
    }
    onChange(next);
  };

  const applyList = () => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const selected = value.slice(start, end) || "item";
    const listed = selected
      .split("\n")
      .map((line) => (line.startsWith("- ") ? line : `- ${line}`))
      .join("\n");
    const next = `${value.slice(0, start)}${listed}${value.slice(end)}`;
    if (next.length > maxLength) {
      return;
    }
    onChange(next);
  };

  return (
    <div className="mb-2 flex items-center gap-0.5" data-testid="work-order-description-toolbar">
      <FormatButton label="Bold" disabled={disabled} onMouseDown={() => applyWrap("**")}>
        <Bold className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton label="Italic" disabled={disabled} onMouseDown={() => applyWrap("_")}>
        <Italic className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton label="Code" disabled={disabled} onMouseDown={() => applyWrap("`")}>
        <Code className="size-3.5" aria-hidden />
      </FormatButton>
      <FormatButton label="List" disabled={disabled} onMouseDown={applyList}>
        <List className="size-3.5" aria-hidden />
      </FormatButton>
    </div>
  );
}

function FormatButton({
  label,
  disabled,
  onMouseDown,
  children,
}: {
  label: string;
  disabled: boolean;
  onMouseDown: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={label}
      disabled={disabled}
      className="size-7 text-muted-foreground"
      onMouseDown={(event) => {
        event.preventDefault();
        onMouseDown();
      }}
    >
      {children}
    </Button>
  );
}
