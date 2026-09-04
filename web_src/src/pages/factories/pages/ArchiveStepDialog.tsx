import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

type ArchiveStepDialogProps = {
  open: boolean;
  stepName: string;
  hasTasks: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
};

export function ArchiveStepDialog({ open, stepName, hasTasks, onOpenChange, onConfirm }: ArchiveStepDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent data-testid="lines-archive-step-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{hasTasks ? "Archive this step" : "Archive this step?"}</AlertDialogTitle>
          <AlertDialogDescription>
            {hasTasks
              ? "Make sure that the column does not have any tasks in it."
              : `This removes ${stepName} from the line. The automation stays in the workspace.`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          {hasTasks ? (
            <AlertDialogCancel data-testid="lines-archive-step-close">Close</AlertDialogCancel>
          ) : (
            <>
              <AlertDialogCancel data-testid="lines-archive-step-cancel">Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={onConfirm} data-testid="lines-archive-step-confirm">
                Archive
              </AlertDialogAction>
            </>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
