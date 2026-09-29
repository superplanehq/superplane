import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { usePermissions } from "@/contexts/usePermissions";
import { useAccount } from "@/contexts/useAccount";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";

import { FactoriesLayout } from "../layout/FactoriesLayout";
import { WorkspaceLoadingProvider } from "../layout/workspaceLoading";
import { LinesPage } from "./LinesPage";
import { OnboardingGate } from "./onboarding/OnboardingGate";
import { PublicFactoryBoardPage } from "./PublicFactoryBoardPage";

export function FactoryLineAccessGate() {
  const { account, loading } = useAccount();

  if (loading) {
    return <div className="flex h-screen items-center justify-center" data-testid="factory-line-access-loading" />;
  }

  if (!account) {
    return <PublicFactoryBoardPage signedIn={false} />;
  }

  return <MemberLineGate />;
}

function SignedInPublicBoard() {
  const { account } = useAccount();
  return <PublicFactoryBoardPage signedIn accountName={account?.name} accountAvatarUrl={account?.avatar_url} />;
}

function MemberLineGate() {
  const { canAct, isLoading } = usePermissions();

  if (isLoading) {
    return <div className="flex h-screen items-center justify-center" data-testid="factory-line-access-loading" />;
  }

  if (!canAct("factories", "read") || !canAct("work_orders", "read")) {
    return <SignedInPublicBoard />;
  }

  return (
    <WorkspaceLoadingProvider>
      <RequireExperimentalFeature featureId={FEATURE_FACTORIES}>
        <FactoriesLayout>
          <OnboardingGate>
            <LinesPage />
          </OnboardingGate>
        </FactoriesLayout>
      </RequireExperimentalFeature>
    </WorkspaceLoadingProvider>
  );
}
