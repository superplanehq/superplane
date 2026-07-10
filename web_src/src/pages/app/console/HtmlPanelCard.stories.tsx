import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { cn } from "@/lib/utils";

import { HtmlBody, HtmlBodyLoading } from "./HtmlBody";
import { CONSOLE_PANEL_BODY_SURFACE, CONSOLE_PANEL_SHELL_SURFACE } from "./consolePanelStyles";
import { PanelFrame } from "./__stories__/storyDecorators";

/**
 * HTML panel content renderer. Like the markdown panel, the real card resolves
 * `{{ name.field }}` variables through `useMarkdownVariables`, so these stories
 * render the pure `HtmlBody` (which sanitizes + scopes the markup) with static
 * `vars`. `HtmlCardFrame` mirrors the typed-panel chrome.
 */
const meta = {
  title: "Console/Html",
  component: HtmlBody,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
} satisfies Meta<typeof HtmlBody>;

export default meta;
type Story = StoryObj<typeof meta>;

function HtmlCardFrame({ title, children }: { title: string; children: ReactNode }) {
  return (
    <PanelFrame height={320}>
      <div
        className={cn(
          "group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70",
          CONSOLE_PANEL_SHELL_SURFACE,
        )}
      >
        <div className="flex items-center justify-between rounded-t-lg py-1.5 pl-3 pr-1.5">
          <span className="truncate text-[13px] font-medium text-slate-700 dark:text-gray-300" title={title}>
            {title}
          </span>
        </div>
        <div className={cn("min-h-0 flex-1 overflow-auto rounded-b-lg bg-white px-4 py-3", CONSOLE_PANEL_BODY_SURFACE)}>
          {children}
        </div>
      </div>
    </PanelFrame>
  );
}

const richHtml = `
<h2>Release notes</h2>
<p>This week's <strong>production</strong> highlights:</p>
<ul>
  <li>Faster console load</li>
  <li>New chart panel</li>
  <li>Bug fixes</li>
</ul>
<blockquote>Thanks to everyone who shipped this release.</blockquote>
<p><a href="https://example.com/changelog">Full changelog</a></p>
`;

const styledHtml = `
<style>
  .badge { display:inline-block; background:#0284c7; color:#fff; padding:2px 8px; border-radius:9999px; font-size:11px; }
  .grid { display:flex; gap:8px; margin-top:8px; }
  .card { flex:1; border:1px solid #e2e8f0; border-radius:8px; padding:8px; }
</style>
<h3>Status board <span class="badge">live</span></h3>
<div class="grid">
  <div class="card"><strong>API</strong><br/>Healthy</div>
  <div class="card"><strong>Web</strong><br/>Degraded</div>
</div>
`;

const variableHtml = `
<h3>Latest deploy</h3>
<p>Version <strong>{{ deploy.version }}</strong> — status: <code>{{ deploy.status }}</code></p>
`;

export const RichContent: Story = {
  render: () => (
    <HtmlCardFrame title="Release notes">
      <HtmlBody body={richHtml} vars={{}} />
    </HtmlCardFrame>
  ),
};

export const ScopedStyles: Story = {
  render: () => (
    <HtmlCardFrame title="Status board">
      <HtmlBody body={styledHtml} vars={{}} />
    </HtmlCardFrame>
  ),
};

export const WithInterpolatedVariables: Story = {
  render: () => (
    <HtmlCardFrame title="Latest deploy">
      <HtmlBody body={variableHtml} vars={{ deploy: { version: "v1.4.2", status: "passed" } }} />
    </HtmlCardFrame>
  ),
};

export const Loading: Story = {
  render: () => (
    <HtmlCardFrame title="Latest deploy">
      <HtmlBodyLoading />
    </HtmlCardFrame>
  ),
};
