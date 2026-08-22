import { Check, MoreHorizontal, Pencil, XIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { LINE_BOARD_COLUMN_COLORS, type LineBoardColumnColorId } from "./lineBoardColumnColors";

interface ColumnLaneMenuProps {
  title: string;
  testId: string;
  /** Opens the automation canvas. Ignored when onEdit is set. */
  editHref?: string | null;
  /** Opens column settings. Use this for Backlog instead of a canvas href. */
  onEdit?: () => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
}

/**
 * Column header menu: Edit (optional) and a compact colour grid.
 */
export function ColumnLaneMenu({ title, testId, editHref, onEdit, colorId, onColorChange }: ColumnLaneMenuProps) {
  const navigate = useNavigate();
  const canEdit = Boolean(onEdit || editHref);

  const handleEdit = () => {
    if (onEdit) {
      onEdit();
      return;
    }
    if (editHref) {
      navigate(editHref);
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`${title} menu`}
          className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={testId}
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-0 w-32 p-0" data-testid={`${testId}-content`}>
        {canEdit ? (
          <>
            <div className="p-1">
              <DropdownMenuItem onClick={handleEdit} data-testid={`${testId}-edit`}>
                <Pencil className="h-3.5 w-3.5" aria-hidden />
                Edit
              </DropdownMenuItem>
            </div>
            <DropdownMenuSeparator className="my-0" />
          </>
        ) : null}

        <div className="px-2 pb-2 pt-2" data-testid={`${testId}-color-picker`}>
          <DropdownMenuLabel className="px-0 pb-1.5 pt-0 text-[12px] font-medium text-muted-foreground">
            Set color
          </DropdownMenuLabel>
          <div className="grid grid-cols-3 gap-1.5" role="listbox" aria-label={`Color for ${title}`}>
            {LINE_BOARD_COLUMN_COLORS.map((color) => {
              const selected = colorId === color.id;
              return (
                <button
                  key={color.id}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  aria-label={color.label}
                  title={color.label}
                  data-testid={`${testId}-color-${color.id}`}
                  onClick={() => onColorChange(color.id)}
                  className={cn(
                    "relative flex aspect-square w-full items-center justify-center rounded-md ring-1 ring-inset ring-black/10 transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:ring-white/15",
                    color.className,
                    selected && "ring-2 ring-foreground",
                  )}
                >
                  {selected ? <Check className="size-3 text-foreground" aria-hidden strokeWidth={3} /> : null}
                </button>
              );
            })}
          </div>
          {colorId ? (
            <button
              type="button"
              data-testid={`${testId}-color-remove`}
              onClick={() => onColorChange(null)}
              className="mt-2 flex w-full items-center justify-center gap-1 rounded-md border border-border px-1.5 py-1 text-[12px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <XIcon className="size-3" aria-hidden />
              Remove color
            </button>
          ) : null}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
