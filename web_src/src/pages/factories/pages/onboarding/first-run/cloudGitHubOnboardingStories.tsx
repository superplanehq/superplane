import { useFactoriesThemeClass } from "@/pages/factories/lib/useFactoriesThemeClass";
import type { ComponentProps, ReactNode } from "react";

import { FirstRunConnectScreen } from "./FirstRunConnectScreen";
import { FirstRunOnboardingMap } from "./firstRunOnboardingMap";
import { CLOUD_GITHUB_APP_SLUG, CLOUD_GITHUB_STATE, firstRunStoryChrome } from "./firstRunMocks";

const connectChrome = firstRunStoryChrome(1);

export function CloudGitHubConnectStory(props: Partial<ComponentProps<typeof FirstRunConnectScreen>> = {}) {
  return (
    <FirstRunConnectScreen
      chrome={connectChrome}
      githubAppSlug={CLOUD_GITHUB_APP_SLUG}
      githubState={CLOUD_GITHUB_STATE}
      onConnectGitHub={() => console.log("connect GitHub")}
      onUseInstallation={(installation) => console.log("use installation", installation)}
      {...props}
    />
  );
}

export function CloudGitHubSettingsFrame({ children }: { children: ReactNode }) {
  useFactoriesThemeClass();
  return (
    <div className="min-h-screen bg-background px-8 py-10 text-foreground">
      <div className="mx-auto max-w-xl space-y-6">{children}</div>
    </div>
  );
}

export function CloudGitHubPathMap() {
  return <FirstRunOnboardingMap />;
}
