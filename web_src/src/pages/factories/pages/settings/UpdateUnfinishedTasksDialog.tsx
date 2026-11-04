import { Button } from "@/components/ui/button";
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

type RepositoryChange = {
  previousRepository: string;
  nextRepository: string;
  previousBranch: string;
  nextBranch: string;
};

export function UpdateUnfinishedTasksDialog(
  props: RepositoryChange & {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onConfirm: () => void;
    onKeepTasks: () => void;
  },
) {
  return (
    <AlertDialog open={props.open} onOpenChange={props.onOpenChange}>
      <AlertDialogContent data-testid="factory-settings-update-tasks-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Update unfinished tasks</AlertDialogTitle>
          <AlertDialogDescription>{unfinishedTasksDescription(props)}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel data-testid="factory-settings-update-tasks-cancel">Cancel</AlertDialogCancel>
          <Button
            type="button"
            variant="outline"
            data-testid="factory-settings-update-tasks-keep"
            onClick={props.onKeepTasks}
          >
            Save without updating tasks
          </Button>
          <AlertDialogAction onClick={props.onConfirm} data-testid="factory-settings-update-tasks-confirm">
            Update tasks
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function unfinishedTasksDescription(props: RepositoryChange) {
  const previousRepository = props.previousRepository.trim();
  const nextRepository = props.nextRepository.trim();
  const previousBranch = props.previousBranch.trim();
  const repositoryChanged =
    previousRepository !== "" && previousRepository.toLowerCase() !== nextRepository.toLowerCase();
  const move = repositoryChanged
    ? repositoryTaskMove(previousRepository, nextRepository, previousBranch, props.nextBranch)
    : branchTaskMove(previousBranch, props.nextBranch);
  return `${move} Save without updating tasks keeps current tasks on their repository and branch.`;
}

function repositoryTaskMove(
  previousRepository: string,
  nextRepository: string,
  previousBranch: string,
  nextBranch: string,
) {
  if (previousBranch && previousBranch !== nextBranch) {
    return `Unfinished tasks on "${previousRepository}" with no branch, or with "${previousBranch}", will use "${nextRepository}" and "${nextBranch}". Closed tasks stay unchanged.`;
  }
  if (previousBranch) {
    return `Unfinished tasks on "${previousRepository}" will use "${nextRepository}". Closed tasks stay unchanged.`;
  }
  return `Unfinished tasks on "${previousRepository}" with no branch will use "${nextRepository}" and "${nextBranch}". Closed tasks stay unchanged.`;
}

function branchTaskMove(previousBranch: string, nextBranch: string) {
  if (previousBranch) {
    return `Unfinished tasks with no branch, or with "${previousBranch}", will use "${nextBranch}". Closed tasks stay unchanged.`;
  }
  return `Unfinished tasks with no branch will use "${nextBranch}". Closed tasks stay unchanged.`;
}
