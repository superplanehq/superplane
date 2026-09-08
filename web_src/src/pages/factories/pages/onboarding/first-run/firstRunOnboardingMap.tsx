import { useFactoriesThemeClass } from "@/pages/factories/lib/useFactoriesThemeClass";
import type { ReactNode } from "react";

type PathStep = {
  label: string;
  href?: string;
};

const FIRST_RUN = "/?path=/story/factories-pages-first-run--";
const GITHUB = "/?path=/story/factories-pages-cloud-github-app--";
const SETUP = "/?path=/story/factories-pages-setup--";

const PATHS: Array<{ title: string; steps: PathStep[] }> = [
  {
    title: "Welcome",
    steps: [
      { label: "Clickable journey", href: `${FIRST_RUN}journey` },
      { label: "Welcome", href: `${FIRST_RUN}welcome` },
    ],
  },
  {
    title: "Connect GitHub",
    steps: [
      { label: "Connect GitHub", href: `${GITHUB}connect` },
      { label: "Connect GitHub (error)", href: `${GITHUB}connect-error` },
      { label: "GitHub.com opens Install or Request. That page is not in Storybook." },
      { label: "Loading after GitHub", href: `${GITHUB}connect-loading` },
    ],
  },
  {
    title: "Select a GitHub account",
    steps: [
      { label: "Connected accounts. Pick one, or install on another.", href: `${GITHUB}account-picker` },
      { label: "One account is waiting for admin approval.", href: `${GITHUB}picker-still-waiting` },
      { label: "Only a waiting request. No connected account yet.", href: `${GITHUB}waiting-named-org` },
      { label: "Waiting request. GitHub has not named the organization yet.", href: `${GITHUB}waiting-unknown-org` },
      { label: "The request is approved. The account is now selectable.", href: `${GITHUB}picker-after-approval` },
      { label: "Binding the selected account", href: `${GITHUB}picker-binding` },
    ],
  },
  {
    title: "Choose a repository",
    steps: [
      { label: "Select a repository", href: `${GITHUB}choose-repository` },
      { label: "Repository is missing. Edit App access on GitHub.", href: `${GITHUB}choose-why-missing` },
      { label: "Loading repositories", href: `${GITHUB}choose-loading` },
    ],
  },
  {
    title: "Connect ticket system",
    steps: [{ label: "Connect ticket system", href: `${FIRST_RUN}tickets` }],
  },
  {
    title: "Connect agent",
    steps: [
      { label: "Hosted credit. This screen is skipped. Tickets finish setup." },
      { label: "Agent with hosted credit", href: `${SETUP}agent-with-grant` },
      { label: "Agent. Connect a provider", href: `${SETUP}agent-without-grant` },
    ],
  },
  {
    title: "After setup",
    steps: [
      { label: "Analysis", href: `${FIRST_RUN}analysis` },
      { label: "Analysis (overrun)", href: `${FIRST_RUN}analysis-overrun` },
      { label: "Analysis (failed)", href: `${FIRST_RUN}analysis-failed` },
      { label: "Board", href: `${FIRST_RUN}board` },
    ],
  },
  {
    title: "Outside this workspace",
    steps: [
      { label: "An admin sees that the request is approved.", href: `${GITHUB}admin-approved` },
      { label: "Organization settings: waiting", href: `${GITHUB}settings-waiting` },
      { label: "Organization settings: waiting (named org)", href: `${GITHUB}settings-waiting-named-org` },
      { label: "Organization settings: account list", href: `${GITHUB}settings-picker` },
      { label: "Organization settings: connection issue", href: `${GITHUB}settings-connection-issue` },
    ],
  },
];

function StoryLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a
      href={href}
      target="_top"
      className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
    >
      {children}
    </a>
  );
}

export function FirstRunOnboardingMap() {
  useFactoriesThemeClass();
  return (
    <div className="min-h-screen bg-background px-8 py-10 text-foreground">
      <article className="mx-auto max-w-2xl space-y-8" data-testid="first-run-onboarding-path-map">
        <header className="space-y-3">
          <h1 className="workspace-page-title font-semibold">Onboarding</h1>
          <p className="text-[15px] leading-6 text-muted-foreground">
            Welcome, connect GitHub, select an account, select a repository, connect tickets, then connect an agent
            when hosted credit is not ready. Account rows can be connected or waiting for approval. You can pick a
            connected account or install the App on another one. Repository rows come from that account.
          </p>
        </header>
        {PATHS.map((path) => (
          <section key={path.title} className="space-y-2">
            <h2 className="text-[15px] font-medium">{path.title}</h2>
            <ol className="list-decimal space-y-1 pl-5 text-[13px] leading-5 text-muted-foreground">
              {path.steps.map((step) => (
                <li key={step.label}>{step.href ? <StoryLink href={step.href}>{step.label}</StoryLink> : step.label}</li>
              ))}
            </ol>
          </section>
        ))}
      </article>
    </div>
  );
}
