import { Link } from "@/components/Link/link";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactory } from "@/hooks/useFactoryData";
import { useOrganizationInviteLink } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useState } from "react";
import { useNavigate, useParams } from "react-router";

import { factoryListPath, factoryOnboardingPath } from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { SetupSections, type SectionId } from "./OnboardingWireframe";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

/**
 * First step of workspace setup, before the workspace exists. The name step
 * creates the workspace and hands over to the setup wizard on its own route.
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
  const { canAct } = usePermissions();
  const setup = useOnboardingSetupState("", { simulateDiscovery: false });
  const [openSection, setOpenSection] = useState<SectionId>("name");
  const createFactory = useCreateFactory(organizationId);
  const canInvite = canAct("members", "create");
  const invite = useOrganizationInviteLink(organizationId, canInvite);
  const inviteUrl =
    invite.data?.enabled && invite.data.token ? `${window.location.origin}/invite/${invite.data.token}` : null;

  const createWorkspace = async () => {
    try {
      // An empty key lets the server derive a free key from the name.
      const factory = await createFactory.mutateAsync({
        name: setup.workspaceName.trim(),
        description: "",
        key: "",
      });
      if (!factory.key) {
        throw new Error("The workspace was created without a key");
      }
      navigate(factoryOnboardingPath(organizationId, factory.key), { replace: true });
      return true;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create workspace"));
      return false;
    }
  };

  return (
    <div className="min-h-screen w-full bg-background text-foreground" data-testid="new-workspace">
      <div className="mx-auto w-full max-w-3xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>
        <p className="mt-1.5 max-w-2xl text-[13px] text-muted-foreground">
          Hand off small engineering work to AI. SuperPlane finds candidate work in your app and backlog, then a coding
          agent opens pull requests. Finish one section to unlock the next.
        </p>

        <div className="mt-8">
          <SetupSections
            setup={setup}
            openSection={openSection}
            setOpenSection={setOpenSection}
            requestConnect={() => undefined}
            onContinueName={createWorkspace}
            onFinish={() => undefined}
            inviteUrl={inviteUrl}
            inviteLoading={invite.isLoading}
            canInvite={canInvite}
            saving={createFactory.isPending}
            lockAfterName
          />
        </div>

        <div className="mt-6">
          <Link href={factoryListPath(organizationId)} className="text-[13px] text-muted-foreground hover:underline">
            Cancel
          </Link>
        </div>
      </div>
    </div>
  );
}
