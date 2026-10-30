import { useAccount } from "@/contexts/useAccount";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { FEATURE_GITHUB_ENTERPRISE } from "@/lib/license";

import { RepositoryBranchPromptLoader } from "../RepositoryBranchPromptLoader";
import type { RepositoryBranchPrompt } from "../repositoryBranchPrompt";
import type { FirstRunSetupFlow, OnboardingPageModel } from "../useFirstRunSetupFlow";
import { selectWorkspaceRepository } from "../workspaceRepositorySelection";
import { FirstRunBitbucketChooseScreen } from "./FirstRunBitbucketChooseScreen";
import { FirstRunBitbucketForgeScreen } from "./FirstRunBitbucketForgeScreen";
import { FirstRunChooseScreen } from "./FirstRunChooseScreen";
import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import { GitHubEnterpriseChooseHost } from "./FirstRunGitHubEnterpriseChooseScreen";
import { FirstRunGitHubEnterpriseHostScreen } from "./FirstRunGitHubEnterpriseHostScreen";
import { githubEnterpriseHostPhase } from "./githubEnterpriseHostPhase";
import { FirstRunHostScreen } from "./FirstRunHostScreen";
import type { FirstRunChrome } from "./firstRunTypes";
import { repositorySphereFor, sphereFor } from "./firstRunSphereFor";

function bitbucketForgePhase(
  identityLinked: boolean,
  repositoryCount: number,
  installedWorkspaceCount: number,
): "connect" | "grant" | "choose" {
  if (!identityLinked) return "connect";
  if (repositoryCount === 0 && installedWorkspaceCount === 0) return "grant";
  return "choose";
}

function BitbucketChooseHost({
  organizationId,
  flow,
  model,
  chrome,
  branchPrompt,
}: {
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
  branchPrompt?: RepositoryBranchPrompt;
}) {
  const connected = model.setup.connected.has("bitbucket");
  const resources = useIntegrationResources(organizationId, model.bitbucketIntegrationId, "repository", undefined, {
    enabled: connected && !flow.bitbucketForgeConfigured,
  });
  const repositories = (resources.data ?? []).map((resource) => resource.name ?? "").filter(Boolean);
  const sphere = connected ? repositorySphereFor(model.setup.selectedRepo) : sphereFor("connect", null);
  if (flow.bitbucketOnboardingPending || flow.bitbucketForgeConfigured || flow.bitbucketLookupFailed) {
    return (
      <FirstRunBitbucketForgeScreen
        phase={bitbucketForgePhase(
          flow.bitbucketIdentityLinked,
          flow.bitbucketRepositories.length,
          flow.bitbucketInstalledWorkspaces.length,
        )}
        connectHref={flow.bitbucketConnectHref}
        repositories={flow.bitbucketRepositories}
        installedWorkspaces={flow.bitbucketInstalledWorkspaces}
        selectedRepository={model.setup.selectedRepo}
        granting={flow.blockingAction === "opening-bitbucket"}
        saving={flow.blockingAction === "saving-repository"}
        loading={flow.bitbucketOnboardingPending}
        loadError={flow.bitbucketLoadError}
        lookupFailed={flow.bitbucketLookupFailed}
        retrying={flow.bitbucketLookupRetrying}
        attemptActive={flow.bitbucketInstallationAttemptActive}
        attemptTimedOut={flow.bitbucketInstallationAttemptTimedOut}
        login={flow.bitbucketLogin}
        providerUserId={flow.bitbucketProviderUserId}
        identities={flow.bitbucketIdentities}
        switchingAccount={flow.blockingAction === "switching-bitbucket-account"}
        switchFailed={flow.bitbucketSwitchFailed}
        chrome={chrome}
        sphere={sphere}
        onRetryLookup={flow.retryBitbucketLookup}
        onGrantAccess={() => void flow.grantBitbucketAccess()}
        installUrl={flow.bitbucketInstallUrl}
        onCopyInstallLink={() => flow.copyBitbucketInstallLink()}
        onSelectRepository={(repository) => selectWorkspaceRepository(model, repository)}
        onClearRepository={model.setup.clearRepository}
        onContinue={() => void flow.continueFromRepository()}
        branchPrompt={branchPrompt}
        onSelectIdentity={(providerUserId) => void flow.selectBitbucketIdentity(providerUserId)}
        onConnectAnotherAccount={() => window.location.assign(flow.bitbucketConnectHref)}
      />
    );
  }
  return (
    <FirstRunBitbucketChooseScreen
      connected={connected}
      repositories={repositories}
      selectedRepository={model.setup.selectedRepo}
      loading={resources.isLoading}
      loadError={resources.isError}
      saving={flow.blockingAction === "saving-repository"}
      chrome={chrome}
      sphere={sphere}
      onConnect={flow.connectBitbucket}
      onSelectRepository={(repository) => selectWorkspaceRepository(model, repository)}
      onContinue={() => void flow.continueFromRepository()}
      branchPrompt={branchPrompt}
    />
  );
}

type FirstRunVcsScreenProps = {
  screen: "host" | "connect" | "ghe-host" | "choose";
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome: FirstRunChrome;
};

export function FirstRunVcsScreen(props: FirstRunVcsScreenProps) {
  const model = props.model;
  if (!model.defaultBranchRequired) return <FirstRunVcsChoices {...props} />;
  return (
    <RepositoryBranchPromptLoader
      organizationId={props.organizationId}
      integrationId={model.branchIntegrationId}
      repository={model.setup.selectedRepo}
      branch={model.chosenDefaultBranch}
      onChange={model.setChosenDefaultBranch}
      render={(branchPrompt) => <FirstRunVcsChoices {...props} branchPrompt={branchPrompt} />}
    />
  );
}

function FirstRunVcsChoices({
  screen,
  organizationId,
  flow,
  model,
  chrome,
  branchPrompt,
}: FirstRunVcsScreenProps & { branchPrompt?: RepositoryBranchPrompt }) {
  const { account } = useAccount();
  const githubEnterpriseLicensed = Boolean(account?.license?.features?.includes(FEATURE_GITHUB_ENTERPRISE));
  const setup = model.setup;
  if (screen === "host") {
    return (
      <FirstRunHostScreen
        selectedHost={flow.vcsHost}
        githubEnterpriseSelected={flow.githubEnterprise}
        githubEnterpriseLicensed={githubEnterpriseLicensed}
        chrome={chrome}
        sphere={sphereFor("host", setup.selectedRepo)}
        onChooseHost={flow.chooseVcsHost}
      />
    );
  }

  if (screen === "ghe-host") {
    const phase = githubEnterpriseHostPhase({
      integrationId: flow.githubEnterpriseIntegrationId,
      ready: flow.githubEnterpriseReady,
      currentStepName: flow.githubEnterpriseCurrentStepName,
      showHost: flow.showGitHubEnterpriseHost,
    });
    const hostChrome = phase === "organization" ? { ...chrome, onBack: flow.returnToGitHubEnterpriseHost } : chrome;
    return (
      <FirstRunGitHubEnterpriseHostScreen
        initialHost={flow.githubEnterpriseHost}
        connecting={flow.blockingAction === "opening-github"}
        phase={phase}
        connectError={flow.githubEnterpriseError}
        chrome={hostChrome}
        sphere={sphereFor("connect", setup.selectedRepo)}
        onContinueHost={(host) => void flow.connectGitHubEnterprise(host)}
        onContinueOrganization={(owner) => void flow.connectGitHubEnterpriseOrganization(owner)}
        onResumeSetup={() => void flow.resumeGitHubEnterpriseSetup()}
      />
    );
  }

  if (screen === "connect") {
    return (
      <FirstRunConnectScreen
        loading={flow.repositoriesLoading}
        connecting={flow.blockingAction === "opening-github"}
        createApp={!flow.appConfigured}
        connectError={flow.connectError}
        chrome={chrome}
        sphere={sphereFor("connect", setup.selectedRepo)}
        onConnectGitHub={() => void (flow.appConfigured ? flow.connectGitHub() : flow.createGitHubApp())}
      />
    );
  }

  if (flow.vcsHost === "bitbucket") {
    return (
      <BitbucketChooseHost
        organizationId={organizationId}
        flow={flow}
        model={model}
        chrome={chrome}
        branchPrompt={branchPrompt}
      />
    );
  }

  if (flow.githubEnterprise) {
    return (
      <GitHubEnterpriseChooseHost
        organizationId={organizationId}
        flow={flow}
        model={model}
        chrome={chrome}
        branchPrompt={branchPrompt}
      />
    );
  }

  return (
    <FirstRunChooseScreen
      repositories={flow.repositories}
      selectedRepository={setup.selectedRepo}
      githubLogin={flow.githubLogin}
      githubUserId={flow.githubUserId}
      githubIdentities={flow.githubIdentities}
      loading={flow.repositoriesLoading}
      saving={flow.blockingAction === "saving-repository"}
      grantingAccess={flow.blockingAction === "opening-github"}
      switchingGitHubAccount={flow.blockingAction === "switching-github-account"}
      synchronizing={flow.synchronizing}
      pendingOrganizations={flow.pendingOrganizations}
      appConfigured={flow.appConfigured}
      chrome={chrome}
      sphere={sphereFor("choose", setup.selectedRepo, model.githubOwner)}
      onSelectRepository={(repository) => selectWorkspaceRepository(model, repository)}
      branchPrompt={branchPrompt}
      onClearRepository={setup.clearRepository}
      onSelectGitHubIdentity={(userId) => void flow.selectGitHubIdentity(userId)}
      onConnectAnotherGitHubAccount={() => void flow.connectGitHub()}
      onGrantAccess={() => void flow.grantGitHubAccess()}
      onContinue={() => void flow.continueFromRepository()}
    />
  );
}
