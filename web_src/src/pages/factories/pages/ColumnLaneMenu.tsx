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
  /** Opens the automation editor when present. */
  editHref?: string | null;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
}

/**
 * Column header menu: Edit (optional) and circular color swatches
 * inspired by Apple Notes / HoneyBook tag pickers.
 */
export function ColumnLaneMenu({ title, testId, editHref, colorId, onColorChange }: ColumnLaneMenuProps) {
  const navigate = useNavigate();

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
      <DropdownMenuContent align="end" className="w-[13.5rem] p-0" data-testid={`${testId}-content`}>
        {editHref ? (
          <>
            <div className="p-1">
              <DropdownMenuItem onClick={() => navigate(editHref)} data-testid={`${testId}-edit`}>
                <Pencil className="h-3.5 w-3.5" aria-hidden />
                Edit
              </DropdownMenuItem>
            </div>
            <DropdownMenuSeparator className="my-0" />
          </>
        ) : null}

        <div className="px-3 pb-2 pt-2.5" data-testid={`${testId}-color-picker`}>
          <DropdownMenuLabel className="px-0 pb-2 pt-0 text-[12px] font-medium text-muted-foreground">
            Set color
          </DropdownMenuLabel>
          <div className="grid grid-cols-5 gap-2" role="listbox" aria-label={`Color for ${title}`}>
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
                    "relative flex size-6 items-center justify-center rounded-full transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    color.swatchClassName,
                    selected && "ring-2 ring-foreground/80 ring-offset-2 ring-offset-background",
                  )}
                >
                  {selected ? <Check className="size-3 text-white drop-shadow-sm" aria-hidden strokeWidth={3} /> : null}
                </button>
              );
            })}
          </div>
          {colorId ? (
            <button
              type="button"
              data-testid={`${testId}-color-remove`}
              onClick={() => onColorChange(null)}
              className="mt-2.5 flex w-full items-center justify-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-[12px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
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
