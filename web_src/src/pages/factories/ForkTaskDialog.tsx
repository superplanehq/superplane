import { useId, useState } from "react";
import { useNavigate } from "react-router";

import type { ForkWorkOrderRequestMode } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useForkWorkOrder } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast } from "@/lib/toast";

import { workOrderDetailPath } from "./lib/factoryPagePaths";
import { FORK_TASK_COPY } from "./lib/forkTask";

export type ForkTaskTarget = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  orderId: string;
  hasPlan: boolean;
  canFork: boolean;
};

export function ForkTaskDialog({
  open,
  onOpenChange,
  target,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: ForkTaskTarget;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="fork-task-dialog">
        <DialogHeader>
          <DialogTitle>{FORK_TASK_COPY.title}</DialogTitle>
          <DialogDescription>{FORK_TASK_COPY.helper}</DialogDescription>
        </DialogHeader>
        {open ? <ForkTaskForm target={target} onDone={() => onOpenChange(false)} /> : null}
      </DialogContent>
    </Dialog>
  );
}

function ForkTaskForm({ target, onDone }: { target: ForkTaskTarget; onDone: () => void }) {
  const intakeHelpId = useId();
  const planHelpId = useId();
  const [mode, setMode] = useState<ForkWorkOrderRequestMode>("MODE_INTAKE");
  const forkTask = useForkWorkOrder(target.organizationId, target.factoryId);
  const navigate = useNavigate();
  const planDisabled = !target.hasPlan || !target.canFork;

  const submit = async () => {
    if (!target.canFork) {
      return;
    }
    try {
      const order = await forkTask.mutateAsync({ orderId: target.orderId, mode });
      onDone();
      if (order.number != null) {
        navigate(workOrderDetailPath(target.organizationId, target.factoryKey, order.number));
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, FORK_TASK_COPY.error));
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <div role="radiogroup" aria-label={FORK_TASK_COPY.title} className="flex flex-col gap-2">
        <ForkChoice
          label={FORK_TASK_COPY.intake}
          helper={FORK_TASK_COPY.intakeHelper}
          helperId={intakeHelpId}
          checked={mode === "MODE_INTAKE"}
          disabled={!target.canFork}
          onSelect={() => setMode("MODE_INTAKE")}
        />
        <ForkChoice
          label={FORK_TASK_COPY.plan}
          helper={target.hasPlan ? FORK_TASK_COPY.planHelper : FORK_TASK_COPY.noPlan}
          helperId={planHelpId}
          checked={mode === "MODE_PLAN"}
          disabled={planDisabled}
          onSelect={() => setMode("MODE_PLAN")}
        />
      </div>
      {target.canFork ? null : (
        <p className="text-sm text-muted-foreground" data-testid="fork-task-permission">
          {FORK_TASK_COPY.permission}
        </p>
      )}
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          {FORK_TASK_COPY.cancel}
        </Button>
        <Button
          type="button"
          onClick={() => void submit()}
          disabled={!target.canFork || forkTask.isPending}
          data-testid="fork-task-submit"
        >
          {FORK_TASK_COPY.submit}
        </Button>
      </DialogFooter>
    </div>
  );
}

function ForkChoice({
  label,
  helper,
  helperId,
  checked,
  disabled,
  onSelect,
}: {
  label: string;
  helper: string;
  helperId: string;
  checked: boolean;
  disabled: boolean;
  onSelect: () => void;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      role="radio"
      aria-label={label}
      aria-checked={checked}
      aria-describedby={helperId}
      disabled={disabled}
      className={cn(
        "h-auto flex-col items-start gap-1 px-3 py-2 text-left font-normal",
        checked && "border-foreground",
      )}
      onClick={onSelect}
    >
      <span className="font-medium">{label}</span>
      <span id={helperId} className="text-sm text-muted-foreground">
        {helper}
      </span>
    </Button>
  );
}
