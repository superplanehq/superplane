import type { Meta, StoryObj } from "@storybook/react";
import { BellRing } from "lucide-react";

import { Button } from "../button";

import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./index";

const notifications = [
  {
    title: "Your call has been confirmed.",
    description: "1 hour ago",
  },
  {
    title: "You have a new message!",
    description: "1 hour ago",
  },
  {
    title: "Your subscription is expiring soon!",
    description: "2 hours ago",
  },
];

const meta = {
  title: "shadcn Primitives/Card",
  component: Card,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    className: {
      control: { type: "text" },
    },
  },
  args: {
    className: "w-96",
  },
} satisfies Meta<typeof Card>;

export default meta;

type Story = StoryObj<typeof Card>;

const renderCard = (args: React.ComponentProps<typeof Card>) => (
  <Card {...args}>
    <CardHeader>
      <CardTitle>Notifications</CardTitle>
      <CardDescription>You have 3 unread messages.</CardDescription>
    </CardHeader>
    <CardContent className="grid gap-4">
      {notifications.map((notification) => (
        <div key={notification.title} className="flex items-center gap-4">
          <BellRing className="h-6 w-6" />
          <div>
            <p>{notification.title}</p>
            <p className="text-foreground/60">{notification.description}</p>
          </div>
        </div>
      ))}
    </CardContent>
    <CardFooter>
      <Button variant="link">Close</Button>
    </CardFooter>
  </Card>
);

export const Default: Story = {
  render: renderCard,
};
