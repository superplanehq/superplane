import { TimeAgo } from "@/components/TimeAgo";
import { Button as UIButton } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Pencil, Trash2 } from "lucide-react";

export function EnterEditDraftDropdown({
  onContinueEditing,
  onDiscardAndStartEdit,
  updatedAt,
}: {
  onContinueEditing: () => void;
  onDiscardAndStartEdit: () => void;
  updatedAt?: string;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <UIButton type="button" variant="default" size="sm" data-testid="canvas-edit-button">
          <Pencil className="h-3.5 w-3.5" />
          Edit
        </UIButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72 px-2">
        <div className="px-3 pt-3 pb-2">
          <div className="text-sm font-medium text-slate-900">You have an unpublished draft</div>
          {updatedAt ? (
            <div className="text-xs text-slate-500">
              Last edited <TimeAgo date={updatedAt} />
            </div>
          ) : null}
        </div>
        <DropdownMenuSeparator className="my-0" />
        <div className="py-1">
          <DropdownMenuItem onClick={onContinueEditing} className="gap-2 px-3 py-2 text-slate-700 cursor-pointer">
            <Pencil className="h-4 w-4" />
            <span>Continue editing</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={onDiscardAndStartEdit}
            className="gap-2 px-3 py-2 text-red-600 focus:text-red-700 cursor-pointer"
          >
            <Trash2 className="h-4 w-4" />
            <span>Discard draft &amp; start fresh</span>
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
