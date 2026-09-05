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
  /** True when this is the only remaining line step. */
  isLastStep?: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
};

export function ArchiveStepDialog({
  open,
  stepName,
  hasTasks,
  isLastStep = false,
  onOpenChange,
  onConfirm,
}: ArchiveStepDialogProps) {
  const blocked = hasTasks || isLastStep;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent data-testid="lines-archive-step-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{blocked ? "Archive this step" : "Archive this step?"}</AlertDialogTitle>
          <AlertDialogDescription>
            {hasTasks
              ? "Make sure that the column does not have any tasks in it."
              : isLastStep
                ? "A line must have at least one step."
                : `This archives ${stepName} and the automation.`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          {blocked ? (
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
