import type { Meta, StoryObj } from "@storybook/react";
import { Check, Loader2, Settings } from "lucide-react";

import { Button } from "../button";

import {
  ContextMenu,
  ContextMenuCheckboxItem,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuPortal,
  ContextMenuRadioGroup,
  ContextMenuRadioItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuSub,
  ContextMenuSubContent,
  ContextMenuSubTrigger,
  ContextMenuTrigger,
} from "./index";

type ContextMenuArgs = React.ComponentProps<typeof ContextMenu>;

const meta = {
  title: "shadcn Primitives/ContextMenu",
  component: ContextMenu,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    modal: {
      control: { type: "boolean" },
    },
  },
  args: {
    modal: false,
  },
} satisfies Meta<ContextMenuArgs>;

export default meta;

type Story = StoryObj<ContextMenuArgs>;

const BaseMenu = (args: ContextMenuArgs) => (
  <ContextMenu {...args}>
    <ContextMenuTrigger asChild>
      <Button variant="outline">Right click here</Button>
    </ContextMenuTrigger>
    <ContextMenuContent className="w-64">
      <ContextMenuLabel>Quick actions</ContextMenuLabel>
      <ContextMenuItem>
        Profile
        <ContextMenuShortcut>⌘P</ContextMenuShortcut>
      </ContextMenuItem>
      <ContextMenuItem>
        Settings
        <ContextMenuShortcut>⌘,</ContextMenuShortcut>
      </ContextMenuItem>
      <ContextMenuSub>
        <ContextMenuSubTrigger inset>Theme</ContextMenuSubTrigger>
        <ContextMenuPortal>
          <ContextMenuSubContent>
            <ContextMenuRadioGroup value="system">
              <ContextMenuRadioItem value="light">Light</ContextMenuRadioItem>
              <ContextMenuRadioItem value="dark">Dark</ContextMenuRadioItem>
              <ContextMenuRadioItem value="system">System</ContextMenuRadioItem>
            </ContextMenuRadioGroup>
          </ContextMenuSubContent>
        </ContextMenuPortal>
      </ContextMenuSub>
      <ContextMenuSeparator />
      <ContextMenuCheckboxItem checked>Show status bar</ContextMenuCheckboxItem>
      <ContextMenuCheckboxItem>Show sidebar</ContextMenuCheckboxItem>
    </ContextMenuContent>
  </ContextMenu>
);

export const Default: Story = {
  render: (args) => <BaseMenu {...args} />,
};

export const WithIndicators: Story = {
  render: (args) => (
    <ContextMenu {...args}>
      <ContextMenuTrigger asChild>
        <Button>
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          Processing...
        </Button>
      </ContextMenuTrigger>
      <ContextMenuContent className="w-72">
        <ContextMenuLabel>Automation</ContextMenuLabel>
        <ContextMenuItem>
          <Settings className="mr-2 h-4 w-4" />
          Configure
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuCheckboxItem checked>
          Enable alerts
          <ContextMenuShortcut>
            <Check className="h-3.5 w-3.5" />
          </ContextMenuShortcut>
        </ContextMenuCheckboxItem>
        <ContextMenuCheckboxItem>Track KPIs</ContextMenuCheckboxItem>
      </ContextMenuContent>
    </ContextMenu>
  ),
};
