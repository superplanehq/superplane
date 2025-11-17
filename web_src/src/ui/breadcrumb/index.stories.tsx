import type { Meta, StoryObj } from "@storybook/react";
import { ArrowRightSquare, MoreHorizontal } from "lucide-react";

import {
  Breadcrumb,
  BreadcrumbEllipsis,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "./index";

type BreadcrumbArgs = React.ComponentProps<typeof Breadcrumb>;

const meta = {
  title: "shadcn Primitives/Breadcrumb",
  component: Breadcrumb,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    className: {
      control: { type: "text" },
    },
  },
} satisfies Meta<BreadcrumbArgs>;

export default meta;

type Story = StoryObj<BreadcrumbArgs>;

const renderTrail = (children: React.ReactNode) => (
  <Breadcrumb>
    <BreadcrumbList>{children}</BreadcrumbList>
  </Breadcrumb>
);

export const Default: Story = {
  render: () =>
    renderTrail(
      <>
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Home</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Library</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>Data</BreadcrumbPage>
        </BreadcrumbItem>
      </>,
    ),
};

export const WithCustomSeparator: Story = {
  render: () =>
    renderTrail(
      <>
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Home</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator>
          <ArrowRightSquare className="h-4 w-4" />
        </BreadcrumbSeparator>
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Projects</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator>
          <ArrowRightSquare className="h-4 w-4" />
        </BreadcrumbSeparator>
        <BreadcrumbItem>
          <BreadcrumbPage>Roadmap</BreadcrumbPage>
        </BreadcrumbItem>
      </>,
    ),
};

export const WithEllipsis: Story = {
  render: () =>
    renderTrail(
      <>
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Home</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbLink href="#">Applications</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbEllipsis className="text-muted-foreground">
            <MoreHorizontal className="h-4 w-4" />
          </BreadcrumbEllipsis>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        <BreadcrumbItem>
          <BreadcrumbPage>Settings</BreadcrumbPage>
        </BreadcrumbItem>
      </>,
    ),
};
