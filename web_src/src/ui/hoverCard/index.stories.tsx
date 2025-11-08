import type { Meta, StoryObj } from "@storybook/react";
import { CalendarDays } from "lucide-react";

import { Avatar, AvatarFallback, AvatarImage } from "../avatar";
import { Button } from "../button";

import { HoverCard, HoverCardContent, HoverCardTrigger } from "./index";

const meta = {
  title: "shadcn Primitives/HoverCard",
  component: HoverCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    openDelay: {
      control: { type: "number", min: 0, max: 1000, step: 50 },
    },
    closeDelay: {
      control: { type: "number", min: 0, max: 1000, step: 50 },
    },
  },
  args: {
    openDelay: 200,
    closeDelay: 200,
  },
  render: (args) => (
    <HoverCard {...args}>
      <HoverCardTrigger asChild>
        <Button variant="link">@superplanehq</Button>
      </HoverCardTrigger>
      <HoverCardContent className="w-80">
        <div className="flex justify-between space-x-4">
          <Avatar>
            <AvatarImage src="https://github.com/superplanehq.png" alt="Superplane" />
            <AvatarFallback>SP</AvatarFallback>
          </Avatar>
          <div className="space-y-1">
            <h4 className="text-sm font-semibold">@superplanehq</h4>
            <p className="text-sm">Superplane UI components and patterns to build workflows faster.</p>
            <div className="flex items-center pt-2 text-xs text-muted-foreground">
              <CalendarDays className="mr-2 h-4 w-4 opacity-70" />
              Joined March 2023
            </div>
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  ),
} satisfies Meta<typeof HoverCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
