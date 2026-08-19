import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button as UIButton } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { softwareFactoryAgentSuggestions } from "@/pages/app/__fixtures__/softwareFactoryAgentSuggestions";
import { Sparkle } from "lucide-react";
import { useState } from "react";

import { AgentSuggestionsHoverCard } from "./AgentSuggestionsHoverCard";
import {
  canvasSidebarToggleActiveClassName,
  canvasSidebarToggleInactiveClassName,
} from "./canvasSidebarToggleClassNames";

const meta = {
  title: "Canvas/AgentSuggestionsHoverCard",
  component: AgentSuggestionsHoverCard,
  parameters: {
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <div className="flex min-h-[280px] items-start justify-center pt-16">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof AgentSuggestionsHoverCard>;

export default meta;

type Story = StoryObj<typeof meta>;

function AgentTriggerButton({ active = false }: { active?: boolean }) {
  return (
    <UIButton
      type="button"
      variant="ghost"
      size={null}
      className={cn(
        "h-7 min-h-7 gap-1.5 rounded-full border-0 py-1 pl-2.5 pr-4 text-[13px] shadow-none transition-colors",
        active ? canvasSidebarToggleActiveClassName : canvasSidebarToggleInactiveClassName,
      )}
    >
      <Sparkle className={cn("size-3.5 shrink-0", active && "text-violet-800")} />
      <span className={cn("text-[13px] font-medium whitespace-nowrap", active && "text-violet-800")}>Agent</span>
    </UIButton>
  );
}

export const Default: Story = {
  args: {
    suggestions: softwareFactoryAgentSuggestions,
    open: true,
    children: <AgentTriggerButton />,
    onSelectSuggestion: (suggestion) => console.log("Select suggestion:", suggestion),
  },
};

export const SingleSuggestion: Story = {
  args: {
    suggestions: [softwareFactoryAgentSuggestions[0]],
    open: true,
    children: <AgentTriggerButton />,
    onSelectSuggestion: (suggestion) => console.log("Select suggestion:", suggestion),
  },
};

export const Empty: Story = {
  args: {
    suggestions: [],
    children: <AgentTriggerButton />,
  },
};

export const OpenByDefault: Story = {
  render: () => {
    const [open, setOpen] = useState(true);
    return (
      <AgentSuggestionsHoverCard
        suggestions={softwareFactoryAgentSuggestions}
        open={open}
        onOpenChange={setOpen}
        onSelectSuggestion={(suggestion) => console.log("Select suggestion:", suggestion)}
      >
        <AgentTriggerButton />
      </AgentSuggestionsHoverCard>
    );
  },
};

export const AgentSidebarOpen: Story = {
  args: {
    suggestions: softwareFactoryAgentSuggestions.slice(0, 3),
    open: true,
    children: <AgentTriggerButton active />,
    onSelectSuggestion: (suggestion) => console.log("Select suggestion:", suggestion),
  },
};
