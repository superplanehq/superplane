import { Link } from "@/components/Link/link";
import { LoadingButton } from "@/components/ui/loading-button";
import { useCreateFactory, useFactories } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { useNavigate, useParams } from "react-router";
import { Loader2 } from "lucide-react";

import { factoryListPath, factorySetupPath } from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { GithubAppRequiredNotice } from "./GithubAppRequiredNotice";
import { saveWithFreeWorkspaceName } from "./uniqueFactoryName";
import { useGithubAppAvailability } from "./useGithubAppAvailability";
import { useOnboardingStorybook } from "./useOnboardingStorybook";
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
  const storybookOnboarding = useOnboardingStorybook();
  const githubApp = useGithubAppAvailability(organizationId);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [retryingCatalog, setRetryingCatalog] = useState(false);
  const [retryingCreation, setRetryingCreation] = useState(false);
  // Workspace creation must run once per attempt, not on every render.
  const requested = useRef(false);

  useEffect(() => {
    // Setup needs the SuperPlane GitHub App, so the workspace is not created
    // until the integration catalog confirms the app is available.
    if (requested.current || factories.isLoading || !githubApp.resolved || !githubApp.available) return;
    requested.current = true;

    const create = async () => {
      try {
        const factory = await saveWithFreeWorkspaceName({
          name: PLACEHOLDER_WORKSPACE_NAME,
          takenNames: (factories.data ?? []).map((existing) => existing.name ?? ""),
          // An empty key lets the server derive a free key from the name.
          save: (name) => createFactory.mutateAsync({ name, description: "", key: "" }),
        });
        if (!factory.id || !factory.key) {
          throw new Error("The workspace was created without a key");
        }
        // Storybook gates setup on this pending pointer. Production uses the
        // server-backed onboarding record and ignores the storybook context.
        storybookOnboarding?.beginOnboarding({
          workspaceId: factory.id,
          workspaceName: factory.name ?? "",
        });
        navigate(factorySetupPath(organizationId, factory.key), { replace: true });
      } catch (creationError) {
        setError(getApiErrorMessage(creationError, "Failed to create workspace"));
      } finally {
        setRetryingCreation(false);
      }
    };

    void create();
  }, [
    attempt,
    createFactory,
    factories.data,
    factories.isLoading,
    githubApp.available,
    githubApp.resolved,
    navigate,
    organizationId,
    storybookOnboarding,
  ]);

  const retry = () => {
    setError(null);
    requested.current = false;
    setRetryingCreation(true);
    setAttempt((current) => current + 1);
  };

  const retryCatalog = async () => {
    if (retryingCatalog) return;
    setRetryingCatalog(true);
    try {
      await githubApp.retry();
    } finally {
      setRetryingCatalog(false);
    }
  };

  if (githubApp.failed) {
    return <GitHubCatalogFailure organizationId={organizationId} retrying={retryingCatalog} onRetry={retryCatalog} />;
  }

  if (githubApp.resolved && !githubApp.available) {
    return <GithubAppRequiredNotice />;
  }

  return (
    <WorkspaceCreationStatus
      organizationId={organizationId}
      error={error}
      retrying={retryingCreation}
      githubAppResolved={githubApp.resolved}
      onRetry={retry}
    />
  );
}

function NewWorkspaceFrame({ children, busy }: { children: ReactNode; busy?: boolean }) {
  return (
    <div className="min-h-screen w-full bg-background text-foreground" data-testid="new-workspace" aria-busy={busy}>
      <div className="mx-auto w-full max-w-3xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>
        {children}
      </div>
    </div>
  );
}

function GitHubCatalogFailure({
  organizationId,
  retrying,
  onRetry,
}: {
  organizationId: string;
  retrying: boolean;
  onRetry: () => Promise<void>;
}) {
  return (
    <NewWorkspaceFrame busy={retrying || undefined}>
      <div className="mt-6 rounded-lg border border-border p-4">
        <p className="text-[13px] text-destructive">SuperPlane could not check the GitHub App.</p>
        <div className="mt-3 flex items-center gap-3">
          <LoadingButton
            type="button"
            size="sm"
            onClick={() => void onRetry()}
            loading={retrying}
            loadingText="Trying again…"
          >
            Try again
          </LoadingButton>
          <Link
            href={factoryListPath(organizationId)}
            aria-disabled={retrying || undefined}
            tabIndex={retrying ? -1 : undefined}
            onClick={(event) => {
              if (retrying) event.preventDefault();
            }}
            className={`text-[13px] text-muted-foreground hover:underline ${retrying ? "pointer-events-none opacity-50" : ""}`}
          >
            Cancel
          </Link>
        </div>
      </div>
    </NewWorkspaceFrame>
  );
}

function WorkspaceCreationStatus({
  organizationId,
  error,
  retrying,
  githubAppResolved,
  onRetry,
}: {
  organizationId: string;
  error: string | null;
  retrying: boolean;
  githubAppResolved: boolean;
  onRetry: () => void;
}) {
  return (
    <NewWorkspaceFrame busy={!error || retrying || undefined}>
      {error ? (
        <div className="mt-6 rounded-lg border border-border p-4">
          <p className="text-[13px] text-destructive">{error}</p>
          <div className="mt-3 flex items-center gap-3">
            <LoadingButton type="button" size="sm" onClick={onRetry} loading={retrying} loadingText="Trying again…">
              Try again
            </LoadingButton>
            <Link href={factoryListPath(organizationId)} className="text-[13px] text-muted-foreground hover:underline">
              Cancel
            </Link>
          </div>
        </div>
      ) : (
        <p className="mt-6 inline-flex items-center gap-2 text-[13px] text-muted-foreground" role="status">
          <Loader2 className="size-4 animate-spin" aria-hidden />
          {retrying ? "Trying again…" : githubAppResolved ? "Creating workspace…" : "Checking GitHub setup…"}
        </p>
      )}
    </NewWorkspaceFrame>
  );
}
