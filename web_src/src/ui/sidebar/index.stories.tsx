import type { Meta, StoryObj } from "@storybook/react";
import { Calendar, ChevronUp, Home, Inbox, Search, Settings, User2 } from "lucide-react";

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../dropdownMenu";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "./index";

const meta = {
  title: "shadcn Primitives/Sidebar",
  component: Sidebar,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  argTypes: {
    side: {
      control: { type: "radio" },
      options: ["left", "right"],
    },
    variant: {
      control: { type: "radio" },
      options: ["sidebar", "floating", "inset"],
    },
    collapsible: {
      control: { type: "radio" },
      options: ["offcanvas", "icon", "none"],
    },
  },
  args: {
    side: "left",
    variant: "sidebar",
    collapsible: "icon",
  },
  decorators: [
    (Story) => (
      <SidebarProvider>
        <div className="flex min-h-svh">
          <Story />
          <main className="flex-1 p-6">
            <SidebarTrigger />
            <p className="mt-4 text-sm text-muted-foreground">Toggle the sidebar to see the different variants.</p>
          </main>
        </div>
      </SidebarProvider>
    ),
  ],
} satisfies Meta<typeof Sidebar>;

export default meta;

type Story = StoryObj<typeof Sidebar>;

const items = [
  { title: "Home", icon: Home, href: "#" },
  { title: "Inbox", icon: Inbox, href: "#" },
  { title: "Calendar", icon: Calendar, href: "#" },
  { title: "Search", icon: Search, href: "#" },
  { title: "Settings", icon: Settings, href: "#" },
];

export const Simple: Story = {
  render: (args) => (
    <Sidebar {...args}>
      <SidebarHeader />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Application</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild>
                    <a href={item.href}>
                      <item.icon />
                      <span>{item.title}</span>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter />
    </Sidebar>
  ),
};

export const WithFooterMenu: Story = {
  render: (args) => (
    <Sidebar {...args}>
      <SidebarHeader />
      <SidebarContent />
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton>
                  <User2 />
                  <span>Username</span>
                  <ChevronUp className="ml-auto" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent side="top" className="w-(--radix-popper-anchor-width)">
                <DropdownMenuItem>Account</DropdownMenuItem>
                <DropdownMenuItem>Billing</DropdownMenuItem>
                <DropdownMenuItem>Sign out</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  ),
};
