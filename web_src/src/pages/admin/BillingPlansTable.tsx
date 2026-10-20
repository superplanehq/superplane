import { Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { BillingHelp } from "./BillingHelp";
import type { BillingPlan } from "./billingPlansApi";

export function BillingPlansTable({
  plans,
  onEdit,
  onDelete,
}: {
  plans: BillingPlan[];
  onEdit: (plan: BillingPlan) => void;
  onDelete: (plan: BillingPlan) => void;
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
                text="Seats are organization members. Parallel task limits are saved but not enforced."
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
                  {plan.parallel_task_limit ?? "Unlimited"} parallel tasks
                </div>
              </TableCell>
              <TableCell>{plan.features.length}</TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  <Button size="icon-sm" variant="ghost" aria-label={`Edit ${plan.name}`} onClick={() => onEdit(plan)}>
                    <Pencil aria-hidden="true" />
                  </Button>
                  {!plan.polar_product_id && !["none", "trial", "business_legacy"].includes(plan.key) && (
                    <Button
                      size="icon-sm"
                      variant="ghost"
                      aria-label={`Delete ${plan.name}`}
                      onClick={() => onDelete(plan)}
                    >
                      <Trash2 aria-hidden="true" />
                    </Button>
                  )}
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
