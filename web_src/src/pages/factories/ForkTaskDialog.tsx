import { useId, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { useNavigate } from "react-router";

import type { ForkWorkOrderRequestMode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
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
  isPending = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: ForkTaskTarget;
  isPending?: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="fork-task-dialog">
        <DialogHeader>
          <DialogTitle>{FORK_TASK_COPY.title}</DialogTitle>
          <DialogDescription>{FORK_TASK_COPY.helper}</DialogDescription>
        </DialogHeader>
        {open ? <ForkTaskForm target={target} isPending={isPending} onDone={() => onOpenChange(false)} /> : null}
      </DialogContent>
    </Dialog>
  );
}

function ForkTaskForm({
  target,
  isPending,
  onDone,
}: {
  target: ForkTaskTarget;
  isPending: boolean;
  onDone: () => void;
}) {
  const intakeHelpId = useId();
  const planHelpId = useId();
  const intakeRef = useRef<HTMLButtonElement>(null);
  const planRef = useRef<HTMLButtonElement>(null);
  const [mode, setMode] = useState<ForkWorkOrderRequestMode>("MODE_INTAKE");
  const forkTask = useForkWorkOrder(target.organizationId, target.factoryId);
  const navigate = useNavigate();
  const submitting = isPending || forkTask.isPending;
  const intakeDisabled = !target.canFork || submitting;
  const planDisabled = !target.hasPlan || !target.canFork || submitting;

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

  const selectMode = (next: ForkWorkOrderRequestMode) => {
    setMode(next);
    const ref = next === "MODE_INTAKE" ? intakeRef : planRef;
    ref.current?.focus();
  };

  const onChoiceKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const direction = forkChoiceDirection(event.key);
    if (direction == null) {
      return;
    }
    const next = nextEnabledForkMode(mode, direction, (choice) =>
      choice === "MODE_INTAKE" ? intakeDisabled : planDisabled,
    );
    if (!next) {
      return;
    }
    event.preventDefault();
    selectMode(next);
  };

  return (
    <div className="flex flex-col gap-4">
      <div
        role="radiogroup"
        aria-label={FORK_TASK_COPY.title}
        aria-orientation="vertical"
        className="flex flex-col gap-2"
        onKeyDown={onChoiceKeyDown}
      >
        <ForkChoice
          label={FORK_TASK_COPY.intake}
          helper={FORK_TASK_COPY.intakeHelper}
          helperId={intakeHelpId}
          checked={mode === "MODE_INTAKE"}
          disabled={intakeDisabled}
          buttonRef={intakeRef}
          onSelect={() => selectMode("MODE_INTAKE")}
        />
        <ForkChoice
          label={FORK_TASK_COPY.plan}
          helper={target.hasPlan ? FORK_TASK_COPY.planHelper : FORK_TASK_COPY.noPlan}
          helperId={planHelpId}
          checked={mode === "MODE_PLAN"}
          disabled={planDisabled}
          buttonRef={planRef}
          onSelect={() => selectMode("MODE_PLAN")}
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
        <LoadingButton
          type="button"
          onClick={() => void submit()}
          disabled={!target.canFork}
          loading={submitting}
          data-testid="fork-task-submit"
        >
          {FORK_TASK_COPY.submit}
        </LoadingButton>
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
  buttonRef,
  onSelect,
}: {
  label: string;
  helper: string;
  helperId: string;
  checked: boolean;
  disabled: boolean;
  buttonRef: RefObject<HTMLButtonElement | null>;
  onSelect: () => void;
}) {
  return (
    <Button
      ref={buttonRef}
      type="button"
      variant="outline"
      role="radio"
      aria-label={label}
      aria-checked={checked}
      aria-describedby={helperId}
      tabIndex={checked ? 0 : -1}
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

const FORK_MODES = ["MODE_INTAKE", "MODE_PLAN"] as const;

function forkChoiceDirection(key: string): 1 | -1 | null {
  if (key === "ArrowDown" || key === "ArrowRight") {
    return 1;
  }
  if (key === "ArrowUp" || key === "ArrowLeft") {
    return -1;
  }
  return null;
}

function nextEnabledForkMode(
  current: ForkWorkOrderRequestMode,
  direction: 1 | -1,
  isDisabled: (mode: (typeof FORK_MODES)[number]) => boolean,
): (typeof FORK_MODES)[number] | null {
  const enabled = FORK_MODES.filter((mode) => !isDisabled(mode));
  if (enabled.length < 2) {
    return null;
  }
  const index = enabled.findIndex((mode) => mode === current);
  const start = index < 0 ? 0 : index;
  return enabled[(start + direction + enabled.length) % enabled.length];
}
