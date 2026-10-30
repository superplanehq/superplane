import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { BillingHelp } from "./BillingHelp";
import type { BillingPlanListItem } from "./billingPlansApi";

export function BillingPlansTable({
  plans,
  onEdit,
  onDelete,
}: {
  plans: BillingPlanListItem[];
  onEdit: (plan: BillingPlanListItem) => void;
  onDelete: (plan: BillingPlanListItem) => void;
}) {
  return (
    <div className="overflow-hidden rounded-lg border px-6">
      <Table aria-label="Billing plans" bleed>
        <TableHead>
          <TableRow>
            <TableHeader>Plan</TableHeader>
            <TableHeader>Visibility</TableHeader>
            <TableHeader>
              Pricing{" "}
              <BillingHelp label="Pricing" text="The plan price is in US dollars. Usage charges are separate." />
            </TableHeader>
            <TableHeader>
              Limits{" "}
              <BillingHelp
                label="Plan limits"
                text="Seats are organization members. The parallel runner limit counts reserved or running runner jobs across all workspaces and fleets."
              />
            </TableHeader>
            <TableHeader>Features</TableHeader>
            <TableHeader>
              <span className="sr-only">Actions</span>
            </TableHeader>
          </TableRow>
        </TableHead>
        <TableBody>
          {plans.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} className="text-center text-muted-foreground">
                No billing plans. Create a plan to start.
              </TableCell>
            </TableRow>
          )}
          {plans.map((plan) => (
            <TableRow key={plan.id} className="hover:bg-muted/30">
              <TableCell>
                <div className="font-medium">{plan.name}</div>
                <code className="text-xs text-muted-foreground">{plan.key}</code>
              </TableCell>
              <TableCell>
                <Badge variant="outline" className="capitalize">
                  {plan.visibility}
                </Badge>
              </TableCell>
              <TableCell>
                <span className="font-medium">
                  {(plan.price_cents / 100).toLocaleString("en-US", { style: "currency", currency: "USD" })}
                  {plan.per_seat && " per seat"}
                </span>
              </TableCell>
              <TableCell>
                <div>{plan.seat_limit ?? "Unlimited"} seats</div>
                <div className="text-xs text-muted-foreground">
                  Parallel runner limit: {plan.parallel_task_limit ?? "Unlimited"}
                </div>
              </TableCell>
              <TableCell>{plan.features.length}</TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  <Button size="icon-sm" variant="ghost" aria-label={`Edit ${plan.name}`} onClick={() => onEdit(plan)}>
                    <Pencil aria-hidden="true" />
                  </Button>
                  <DeletePlanButton plan={plan} onDelete={onDelete} />
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function DeletePlanButton({
  plan,
  onDelete,
}: {
  plan: BillingPlanListItem;
  onDelete: (plan: BillingPlanListItem) => void;
}) {
  const reason = plan.delete_disabled_reason;
  if (reason) {
    // Disabled buttons do not receive focus. The wrapper receives hover and
    // keyboard focus so that the reason is available to every user.
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            role="group"
            tabIndex={0}
            aria-label={`Delete ${plan.name} is unavailable`}
            className="inline-flex rounded-md focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          >
            <Button size="icon-sm" variant="ghost" aria-label={`Delete ${plan.name}`} disabled>
              <Trash2 aria-hidden="true" />
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent>{reason}</TooltipContent>
      </Tooltip>
    );
  }
  return <ConfirmDeletePlanButton plan={plan} onDelete={onDelete} />;
}

function ConfirmDeletePlanButton({
  plan,
  onDelete,
}: {
  plan: BillingPlanListItem;
  onDelete: (plan: BillingPlanListItem) => void;
}) {
  const [confirmationOpen, setConfirmationOpen] = useState(false);
  const confirmDelete = () => {
    setConfirmationOpen(false);
    onDelete(plan);
  };
  return (
    <Dialog open={confirmationOpen} onOpenChange={setConfirmationOpen}>
      <DialogTrigger asChild>
        <Button size="icon-sm" variant="ghost" aria-label={`Delete ${plan.name}`}>
          <Trash2 aria-hidden="true" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete plan?</DialogTitle>
          <DialogDescription>
            Delete “{plan.name}” ({plan.key})? This removes the plan from the billing catalog.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setConfirmationOpen(false)}>
            Keep plan
          </Button>
          <Button variant="destructive" onClick={confirmDelete}>
            Delete plan
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
