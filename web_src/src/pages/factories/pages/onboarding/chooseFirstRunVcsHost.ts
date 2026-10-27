import type { FirstRunHostChoice, VcsHostId } from "./onboardingFixtures";
import type { FirstRunScreen } from "./firstRunScreens";

export function chooseFirstRunVcsHost(args: {
  host: FirstRunHostChoice;
  githubEnterprise: boolean;
  setup: { clearRepository: () => void; selectVcsHost: (host: VcsHostId) => void };
  setGitHubEnterprise: (enabled: boolean, host?: string) => void;
  goToScreen: (screen: FirstRunScreen) => void;
}): void {
  const { host, githubEnterprise, setup, setGitHubEnterprise, goToScreen } = args;
  if (host === "gitlab") return;
  if (host === "github-enterprise") {
    if (!githubEnterprise) {
      setup.clearRepository();
    }
    setGitHubEnterprise(true);
    setup.selectVcsHost("github");
    goToScreen("ghe-host");
    return;
  }
  if (host === "github" && githubEnterprise) {
    setup.clearRepository();
  }
  setGitHubEnterprise(false);
  setup.selectVcsHost(host);
  goToScreen(host === "bitbucket" ? "choose" : "connect");
}
