import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { useCreateFactory, useFactories } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";

import { factoryListPath, factorySetupPath } from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { saveWithFreeWorkspaceName } from "./uniqueFactoryName";
import { PLACEHOLDER_WORKSPACE_NAME } from "./workspaceNames";

/**
 * Creates the workspace with a placeholder name and opens the setup wizard.
 * The wizard derives the real name from the selected repository.
 */
export function NewWorkspacePage() {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return <NewWorkspacePageContent organizationId={organizationId} />;
}

function NewWorkspacePageContent({ organizationId }: { organizationId: string }) {
  useFactoriesThemeClass();
  usePageTitle(["New workspace"]);

  const navigate = useNavigate();
  const factories = useFactories(organizationId);
  const createFactory = useCreateFactory(organizationId);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  // Workspace creation must run once per attempt, not on every render.
  const requested = useRef(false);

  useEffect(() => {
    if (requested.current || factories.isLoading) return;
    requested.current = true;

    const create = async () => {
      try {
        const factory = await saveWithFreeWorkspaceName({
          name: PLACEHOLDER_WORKSPACE_NAME,
          takenNames: (factories.data ?? []).map((existing) => existing.name ?? ""),
          // An empty key lets the server derive a free key from the name.
          save: (name) => createFactory.mutateAsync({ name, description: "", key: "" }),
        });
        if (!factory.key) {
          throw new Error("The workspace was created without a key");
        }
        navigate(factorySetupPath(organizationId, factory.key), { replace: true });
      } catch (creationError) {
        setError(getApiErrorMessage(creationError, "Failed to create workspace"));
      }
    };

    void create();
  }, [attempt, createFactory, factories.data, factories.isLoading, navigate, organizationId]);

  const retry = () => {
    setError(null);
    requested.current = false;
    setAttempt((current) => current + 1);
  };

  return (
    <div className="min-h-screen w-full bg-background text-foreground" data-testid="new-workspace">
      <div className="mx-auto w-full max-w-3xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>

        {error ? (
          <div className="mt-6 rounded-lg border border-border p-4">
            <p className="text-[13px] text-destructive">{error}</p>
            <div className="mt-3 flex items-center gap-3">
              <Button type="button" size="sm" onClick={retry}>
                Try again
              </Button>
              <Link
                href={factoryListPath(organizationId)}
                className="text-[13px] text-muted-foreground hover:underline"
              >
                Cancel
              </Link>
            </div>
          </div>
        ) : (
          <p className="mt-6 text-[13px] text-muted-foreground">Creating the workspace…</p>
        )}
      </div>
    </div>
  );
}
