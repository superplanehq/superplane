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

type FactoryAppResetConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
};

export function FactoryAppResetConfirmDialog({ open, onOpenChange, onConfirm }: FactoryAppResetConfirmDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent data-testid="factory-app-reset-defaults-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Reset to factory defaults?</AlertDialogTitle>
          <AlertDialogDescription>
            This replaces your current draft with the bundled template for this app. It keeps your connected repository
            and integrations. The change applies only after you click Save.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel data-testid="factory-app-reset-defaults-cancel">Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} data-testid="factory-app-reset-defaults-confirm">
            Reset to factory defaults
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
