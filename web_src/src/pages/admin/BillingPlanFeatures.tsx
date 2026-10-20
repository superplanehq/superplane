import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { showErrorToast } from "@/lib/toast";
import { billingCatalogRequest, type FeatureDefinition } from "./billingPlansApi";

export function BillingPlanFeatures({
  features,
  onSaved,
}: {
  features: FeatureDefinition[];
  onSaved: () => Promise<void>;
}) {
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
    <section aria-label="Feature catalog" className="space-y-4 rounded border p-4">
      <h2 className="font-semibold">Feature catalog</h2>
      <p>Define features here, then select them in a plan. Use the string key to check feature access in code.</p>
      <Button
        onClick={() => {
          setCreating(true);
          setEditing({ key: "", label: "", gated: true });
        }}
      >
        Create feature
      </Button>
      {features.length === 0 && <p>No features. Create a feature to add it to plans.</p>}
      {features.map((feature) => (
        <div key={feature.key} className="flex items-center gap-3">
          <span className="grow">
            {feature.label} · <code>{feature.key}</code>
          </span>
          <Button
            variant="outline"
            onClick={() => {
              setCreating(false);
              setEditing({ ...feature });
            }}
          >
            Edit {feature.label}
          </Button>
        </div>
      ))}
      {editing && (
        <form
          className="space-y-3"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <div>
            <Label htmlFor="feature-key">Feature key</Label>
            <Input
              id="feature-key"
              value={editing.key}
              disabled={!creating || saving}
              required
              pattern="[a-z][a-z0-9_]*"
              onChange={(event) => setEditing({ ...editing, key: event.target.value })}
            />
            <p>
              Use lowercase letters, numbers, and underscores. Start with a letter. The key cannot change after
              creation.
            </p>
          </div>
          <div>
            <Label htmlFor="feature-label">Feature name</Label>
            <Input
              id="feature-label"
              value={editing.label}
              required
              disabled={saving}
              onChange={(event) => setEditing({ ...editing, label: event.target.value })}
            />
          </div>
          <Label className="flex items-center gap-2">
            <Checkbox
              checked={editing.gated}
              disabled={saving}
              onChange={(event) => setEditing({ ...editing, gated: event.target.checked })}
            />
            Require plan selection or an organization grant
          </Label>
          <p>If cleared, all organizations have access. Changes apply immediately.</p>
          <div className="flex gap-2">
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Save feature"}
            </Button>
            <Button type="button" variant="outline" disabled={saving} onClick={() => setEditing(null)}>
              Cancel
            </Button>
          </div>
        </form>
      )}
    </section>
  );
}
