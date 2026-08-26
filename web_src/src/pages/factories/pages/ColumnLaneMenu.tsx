import { Bot, Check, MoreHorizontal, Pencil, SlidersHorizontal, XIcon } from "lucide-react";
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

import { DEFAULT_LINE_STEP_PARALLELISM, setParallelismLabel } from "../lib/factoryLineFormShared";
import { LINE_BOARD_COLUMN_COLORS, type LineBoardColumnColorId } from "./lineBoardColumnColors";

interface ColumnLaneMenuProps {
  title: string;
  testId: string;
  /** Opens the automation canvas. Ignored when onEdit is set. */
  editHref?: string | null;
  /** Opens the primary edit surface without route navigation. */
  onEdit?: () => void;
  /** Menu item copy. Defaults to Edit. */
  editLabel?: string;
  /** Opens the inline agent editor. */
  onEditAgent?: () => void;
  /** Opens the parallelism modal for canvas-backed phases. */
  onSetParallelism?: () => void;
  parallelism?: number;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
}

/**
 * Column header menu: Edit (optional) and a single row of colour circles.
 */
export function ColumnLaneMenu({
  title,
  testId,
  editHref,
  onEdit,
  editLabel = "Edit",
  onEditAgent,
  onSetParallelism,
  parallelism = DEFAULT_LINE_STEP_PARALLELISM,
  colorId,
  onColorChange,
}: ColumnLaneMenuProps) {
  const navigate = useNavigate();
  const canEdit = Boolean(onEdit || editHref);
  const hasActions = canEdit || Boolean(onEditAgent || onSetParallelism);

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
      <DropdownMenuContent align="end" className="min-w-32 w-max p-0" data-testid={`${testId}-content`}>
        {hasActions ? (
          <>
            <div className="p-1">
              {canEdit ? (
                <DropdownMenuItem onSelect={handleEdit} data-testid={`${testId}-edit`}>
                  <Pencil className="h-3.5 w-3.5" aria-hidden />
                  {editLabel}
                </DropdownMenuItem>
              ) : null}
              {onEditAgent ? (
                <DropdownMenuItem onSelect={onEditAgent} data-testid={`${testId}-edit-agent`}>
                  <Bot className="h-3.5 w-3.5" aria-hidden />
                  Edit Agent
                </DropdownMenuItem>
              ) : null}
              {onSetParallelism ? (
                <DropdownMenuItem onSelect={onSetParallelism} data-testid={`${testId}-parallelism`}>
                  <SlidersHorizontal className="h-3.5 w-3.5" aria-hidden />
                  {setParallelismLabel(parallelism)}
                </DropdownMenuItem>
              ) : null}
            </div>
            <DropdownMenuSeparator className="my-0" />
          </>
        ) : null}

        <div className="px-2 pb-2 pt-2" data-testid={`${testId}-color-picker`}>
          <DropdownMenuLabel className="px-0 pb-1.5 pt-0 text-[12px] font-medium text-muted-foreground">
            Set color
          </DropdownMenuLabel>
          <div className="flex items-center gap-1" role="listbox" aria-label={`Color for ${title}`}>
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
                    "relative flex size-5 shrink-0 items-center justify-center rounded-full ring-1 ring-inset ring-black/10 transition-transform hover:z-10 hover:scale-125 hover:ring-2 hover:ring-foreground/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:ring-white/15",
                    color.className,
                    selected && "ring-2 ring-foreground",
                  )}
                >
                  {selected ? <Check className="size-2.5 text-foreground" aria-hidden strokeWidth={3} /> : null}
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
