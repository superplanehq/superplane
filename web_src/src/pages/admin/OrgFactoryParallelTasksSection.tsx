import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { parsePositiveWholeNumber } from "@/lib/positiveWholeNumber";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Factory } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

type OrganizationFactorySettings = {
  max_parallel_factory_tasks: number | null;
  installation_default: number;
  effective: number;
};

type FactorySettingsRequest = {
  max_parallel_factory_tasks: number | null;
};

const limitHint = "Enter a whole number of at least 1, or leave the field empty.";

const getErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === "") {
    return fallback;
  }
  return text;
};

// An empty field clears the override. Any other value must be a whole
// number, so the saved limit matches what the administrator typed.
const buildRequestBody = (value: string): FactorySettingsRequest | null => {
  if (value.trim() === "") {
    return { max_parallel_factory_tasks: null };
  }

  const limit = parsePositiveWholeNumber(value);
  if (limit == null) {
    return null;
  }

  return { max_parallel_factory_tasks: limit };
};

export function OrgFactoryParallelTasksSection({ orgId }: { orgId: string }) {
  const [settings, setSettings] = useState<OrganizationFactorySettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [value, setValue] = useState("");
  const [saving, setSaving] = useState(false);

  const applySettings = useCallback((data: OrganizationFactorySettings) => {
    setSettings(data);
    setValue(data.max_parallel_factory_tasks == null ? "" : String(data.max_parallel_factory_tasks));
  }, []);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/factory-settings`, {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to load factory settings"));
      }
      applySettings(await response.json());
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load factory settings");
    } finally {
      setLoading(false);
    }
  }, [applySettings, orgId]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const save = async () => {
    const body = buildRequestBody(value);
    if (body == null) {
      showErrorToast(limitHint);
      return;
    }

    setSaving(true);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/factory-settings`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to update factory limit"));
      }
      applySettings(await response.json());
      showSuccessToast("Factory limit updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update factory limit");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Factory size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Factory limits
        </Heading>
      </div>
      {loading && !settings ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">Loading factory limits...</Text>
      ) : settings ? (
        <div>
          <Label className="mb-2 block text-left" htmlFor="org-max-parallel-factory-tasks">
            Maximum parallel factory tasks
          </Label>
          <div className="max-w-sm">
            <Input
              id="org-max-parallel-factory-tasks"
              data-testid="admin-org-max-parallel-factory-tasks"
              value={value}
              onChange={(event) => setValue(event.target.value)}
              placeholder={`Installation default (${settings.installation_default})`}
              inputMode="numeric"
            />
          </div>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Each factory in this organization can run this many tasks at the same time. Leave empty to use the
            installation default.
          </Text>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">Effective limit: {settings.effective}</Text>
          <Button
            type="button"
            className="mt-3"
            data-testid="admin-org-max-parallel-factory-tasks-save"
            onClick={save}
            disabled={saving}
          >
            {saving ? "Saving..." : "Save factory limit"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
