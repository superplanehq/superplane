import type { Meta, StoryObj } from "@storybook/react-vite";

import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";

/**
 * Visual reference for the v3 Cursor-inspired tokens the /factories area
 * consumes. Everything here reads from the same CSS vars as shadcn primitives
 * (`--background`, `--sidebar`, etc.), so hunting for a mismatched hard-coded
 * color is as simple as spotting a swatch that doesn't match.
 */

const SEMANTIC_TOKENS: { name: string; description: string; className: string }[] = [
  { name: "--background", description: "Canvas background", className: "bg-background text-foreground" },
  { name: "--foreground", description: "Primary text", className: "bg-foreground text-background" },
  { name: "--card", description: "Elevated surfaces", className: "bg-card text-card-foreground" },
  { name: "--popover", description: "Dialogs, dropdowns", className: "bg-popover text-popover-foreground" },
  { name: "--primary", description: "Buttons, focus fills", className: "bg-primary text-primary-foreground" },
  {
    name: "--secondary",
    description: "Muted button variant",
    className: "bg-secondary text-secondary-foreground",
  },
  { name: "--muted", description: "Placeholder surfaces", className: "bg-muted text-muted-foreground" },
  { name: "--accent", description: "Hover surfaces", className: "bg-accent text-accent-foreground" },
  {
    name: "--destructive",
    description: "Danger actions",
    className: "bg-destructive text-destructive-foreground",
  },
];

const SIDEBAR_TOKENS: { name: string; description: string; className: string }[] = [
  { name: "--sidebar", description: "Left nav surface", className: "bg-sidebar text-sidebar-foreground" },
  {
    name: "--sidebar-accent",
    description: "Nav row hover / active",
    className: "bg-sidebar-accent text-sidebar-accent-foreground",
  },
];

const OUTLINE_TOKENS: { name: string; description: string; className: string }[] = [
  { name: "--border", description: "Card and divider strokes", className: "border-2 border-border" },
  { name: "--input", description: "Form field strokes", className: "border-2 border-input" },
  { name: "--ring", description: "Focus ring", className: "border-2 border-ring" },
];

const RADIUS_STEPS: { name: string; className: string }[] = [
  { name: "sm (calc(--radius - 4px))", className: "rounded-sm" },
  { name: "md (calc(--radius - 2px))", className: "rounded-md" },
  { name: "lg (--radius)", className: "rounded-lg" },
  { name: "xl (calc(--radius + 4px))", className: "rounded-xl" },
];

const TYPOGRAPHY_STEPS: { name: string; sample: string; className: string }[] = [
  { name: "Settings page title (22 / 600)", sample: "General", className: "workspace-page-title" },
  {
    name: "Section page title (15 / 500)",
    sample: "Overview",
    className: "workspace-section-title font-medium",
  },
  { name: "Section heading (15 / 600)", sample: "Workspace details", className: "workspace-section-title" },
  { name: "Card heading (13 / 500)", sample: "Tasks", className: "text-[13px] font-medium tracking-[-0.01em]" },
  { name: "Body (13 / 400)", sample: "The quick brown fox jumps over the lazy dog.", className: "text-[13px]" },
  {
    name: "Subtitle (13 / 400 muted)",
    sample: "Your workspace at a glance.",
    className: "text-[13px] text-muted-foreground",
  },
  {
    name: "Caption (11 / 500 uppercase)",
    sample: "Recent",
    className: "text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground",
  },
];

interface SwatchProps {
  name: string;
  description?: string;
  className: string;
  label?: string;
}

function Swatch({ name, description, className, label }: SwatchProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <div
        className={`flex h-16 items-center justify-center rounded-md border border-border/60 ${className}`}
        aria-hidden
      >
        {label ? <span className="text-[13px] font-medium">{label}</span> : null}
      </div>
      <div>
        <p className="text-[12px] font-medium text-foreground">{name}</p>
        {description ? <p className="text-[11px] text-muted-foreground">{description}</p> : null}
      </div>
    </div>
  );
}

function SwatchGrid({ title, items }: { title: string; items: SwatchProps[] }) {
  return (
    <section className="space-y-3">
      <h3 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</h3>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {items.map((item) => (
          <Swatch key={item.name} {...item} />
        ))}
      </div>
    </section>
  );
}

function DesignTokensGallery() {
  return (
    <div className="mx-auto max-w-[960px] space-y-10 px-8 py-10">
      <header>
        <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Factories design tokens</h1>
        <p className="mt-1 text-[13px] text-muted-foreground">
          v3 Cursor-inspired palette, radius, and typography scale as consumed by /factories.
        </p>
      </header>

      <SwatchGrid title="Semantic surfaces" items={SEMANTIC_TOKENS} />
      <SwatchGrid title="Sidebar surfaces" items={SIDEBAR_TOKENS} />
      <SwatchGrid title="Outlines &amp; focus" items={OUTLINE_TOKENS} />

      <section className="space-y-3">
        <h3 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Radius scale</h3>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {RADIUS_STEPS.map((step) => (
            <div key={step.name} className="flex flex-col gap-1.5">
              <div
                className={`flex h-16 items-center justify-center border border-border bg-card ${step.className}`}
                aria-hidden
              />
              <p className="text-[12px] font-medium text-foreground">{step.name}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Typography</h3>
        <div className="space-y-4 rounded-lg border border-border bg-card p-6">
          {TYPOGRAPHY_STEPS.map((step) => (
            <div
              key={step.name}
              className="flex flex-col gap-1 border-b border-border/60 pb-4 last:border-b-0 last:pb-0"
            >
              <p className={step.className}>{step.sample}</p>
              <p className="text-[11px] uppercase tracking-[0.04em] text-muted-foreground">{step.name}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

const meta = {
  title: "Factories/Design System/Tokens",
  component: DesignTokensGallery,
  parameters: { layout: "fullscreen" },
  decorators: [withFactoriesTheme],
} satisfies Meta<typeof DesignTokensGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Light: Story = {};

export const Dark: Story = {
  parameters: {
    theme: "dark",
    backgrounds: { default: "dark" },
  },
};
