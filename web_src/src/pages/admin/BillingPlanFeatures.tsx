import { useRef, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "@/components/Table/table";
import { FeatureDeleteDialog } from "./FeatureDeleteDialog";
import { FeatureOrganizationsSheet } from "./FeatureOrganizationsSheet";
import { BillingHelp } from "./BillingHelp";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { showErrorToast } from "@/lib/toast";
import { billingCatalogRequest, type FeatureDefinition } from "./billingPlansApi";

export function BillingPlanFeatures({
  features,
  onSaved,
  refreshErrorFocus,
}: {
  features: FeatureDefinition[];
  onSaved: () => Promise<void>;
  refreshErrorFocus?: () => HTMLElement | null;
}) {
  const returnFocus = useRef<HTMLButtonElement>(null);
  const organizationTrigger = useRef<HTMLButtonElement>(null);
  const createButton = useRef<HTMLButtonElement>(null);
  const [selected, setSelected] = useState<FeatureDefinition | null>(null);
  const [pendingDelete, setPendingDelete] = useState<{ feature: FeatureDefinition; trigger: HTMLButtonElement } | null>(
    null,
  );
  const [editing, setEditing] = useState<FeatureDefinition | null>(null);
  const [creating, setCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const save = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      await billingCatalogRequest(
        `feature-definitions${creating ? "" : `/${editing.key}`}`,
        creating ? "POST" : "PUT",
        editing,
      );
      setEditing(null);
      await onSaved();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to save feature");
    } finally {
      setSaving(false);
    }
  };
  return (
    <section aria-label="Feature catalog" className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <h2 className="font-semibold">Feature catalog</h2>
          <p className="text-sm text-muted-foreground">Define features, then select them in a plan.</p>
        </div>
        <Button
          ref={createButton}
          onClick={(event) => {
            returnFocus.current = event.currentTarget;
            setCreating(true);
            setEditing({ key: "", label: "", gated: true });
          }}
        >
          <Plus aria-hidden="true" />
          Create feature
        </Button>
      </div>
      <FeatureCatalogTable
        features={features}
        onViewOrganizations={(feature, trigger) => {
          organizationTrigger.current = trigger;
          setSelected(feature);
        }}
        onEdit={(feature, trigger) => {
          returnFocus.current = trigger;
          setCreating(false);
          setEditing({ ...feature });
        }}
        onDelete={(feature, trigger) => setPendingDelete({ feature, trigger })}
      />
      {selected && (
        <FeatureOrganizationsSheet
          feature={selected}
          returnFocus={organizationTrigger.current}
          onClose={() => setSelected(null)}
        />
      )}
      {pendingDelete && (
        <FeatureDeleteDialog
          feature={pendingDelete.feature}
          returnFocus={pendingDelete.trigger}
          survivorFocus={() =>
            createButton.current?.isConnected ? createButton.current : (refreshErrorFocus?.() ?? null)
          }
          onDeleted={onSaved}
          onClose={() => setPendingDelete(null)}
        />
      )}
      {editing && (
        <FeatureEditor
          editing={editing}
          creating={creating}
          saving={saving}
          returnFocus={returnFocus.current}
          onChange={setEditing}
          onSave={() => void save()}
          onCancel={() => setEditing(null)}
        />
      )}
    </section>
  );
}

function FeatureCatalogTable({
  features,
  onEdit,
  onDelete,
  onViewOrganizations,
}: {
  features: FeatureDefinition[];
  onEdit: (feature: FeatureDefinition, trigger: HTMLButtonElement) => void;
  onDelete: (feature: FeatureDefinition, trigger: HTMLButtonElement) => void;
  onViewOrganizations: (feature: FeatureDefinition, trigger: HTMLButtonElement) => void;
}) {
  return (
    <div className="overflow-hidden rounded-lg border px-6">
      <Table aria-label="Features" bleed>
        <TableHead>
          <TableRow>
            <TableHeader>Feature</TableHeader>
            <TableHeader>
              Key{" "}
              <BillingHelp
                label="Feature key"
                text="Use this stable identifier to check feature access in code. The key cannot change after creation."
              />
            </TableHeader>
            <TableHeader>
              Access{" "}
              <BillingHelp
                label="Feature access"
                text="Restricted features require plan selection or an organization grant. All organizations have access to unrestricted features."
              />
            </TableHeader>
            <TableHeader>
              <span className="sr-only">Actions</span>
            </TableHeader>
          </TableRow>
        </TableHead>
        <TableBody>
          {features.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} className="text-center text-muted-foreground">
                No features. Create a feature to add it to plans.
              </TableCell>
            </TableRow>
          )}
          {features.map((feature) => (
            <FeatureRow
              key={feature.key}
              feature={feature}
              onViewOrganizations={(trigger) => onViewOrganizations(feature, trigger)}
              onEdit={(trigger) => onEdit(feature, trigger)}
              onDelete={(trigger) => onDelete(feature, trigger)}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function FeatureRow({
  feature,
  onEdit,
  onDelete,
  onViewOrganizations,
}: {
  feature: FeatureDefinition;
  onEdit: (trigger: HTMLButtonElement) => void;
  onDelete: (trigger: HTMLButtonElement) => void;
  onViewOrganizations: (trigger: HTMLButtonElement) => void;
}) {
  return (
    <TableRow className="hover:bg-muted/30">
      <TableCell className="font-medium">{feature.label}</TableCell>
      <TableCell>
        <code className="text-xs text-muted-foreground">{feature.key}</code>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{feature.gated ? "Restricted" : "All organizations"}</Badge>
      </TableCell>
      <TableCell className="text-right">
        <Button size="sm" variant="ghost" onClick={(event) => onViewOrganizations(event.currentTarget)}>
          View organizations
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Edit ${feature.label}`}
          onClick={(event) => onEdit(event.currentTarget)}
        >
          <Pencil aria-hidden="true" />
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label={`Delete ${feature.label}`}
          onClick={(event) => onDelete(event.currentTarget)}
        >
          <Trash2 aria-hidden="true" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

function FeatureEditor({
  editing,
  creating,
  saving,
  returnFocus,
  onChange,
  onSave,
  onCancel,
}: {
  editing: FeatureDefinition;
  creating: boolean;
  saving: boolean;
  returnFocus: HTMLButtonElement | null;
  onChange: (feature: FeatureDefinition) => void;
  onSave: () => void;
  onCancel: () => void;
}) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !saving) onCancel();
      }}
    >
      <DialogContent
        className="max-h-[90vh] overflow-y-auto"
        showCloseButton={!saving}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus?.focus();
        }}
      >
        <DialogHeader>
          <DialogTitle>{creating ? "Create feature" : `Edit ${editing.label}`}</DialogTitle>
          <DialogDescription>Control how organizations receive access to this feature.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            onSave();
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="feature-key">Feature key</Label>
            <Input
              id="feature-key"
              value={editing.key}
              disabled={!creating || saving}
              required
              pattern="[a-z][a-z0-9_]*"
              onChange={(event) => onChange({ ...editing, key: event.target.value })}
            />
            <p className="text-xs leading-relaxed text-muted-foreground">
              Use lowercase letters, numbers, and underscores. Start with a letter. The key cannot change after
              creation.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="feature-label">Feature name</Label>
            <Input
              id="feature-label"
              value={editing.label}
              required
              disabled={saving}
              onChange={(event) => onChange({ ...editing, label: event.target.value })}
            />
          </div>
          <Label className="flex items-center gap-2">
            <Checkbox
              checked={editing.gated}
              disabled={saving}
              onChange={(event) => onChange({ ...editing, gated: event.target.checked })}
            />
            Require plan selection or an organization grant
          </Label>
          <p className="text-xs text-muted-foreground">
            If cleared, all organizations have access. Changes apply immediately.
          </p>
          <DialogFooter className="border-t pt-4">
            <Button type="button" variant="outline" disabled={saving} onClick={() => onCancel()}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Save feature"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
