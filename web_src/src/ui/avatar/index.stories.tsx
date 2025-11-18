import type { Meta, StoryObj } from "@storybook/react";

import { Avatar, AvatarFallback, AvatarImage } from "./index";

type AvatarArgs = React.ComponentProps<typeof Avatar> & {
  src?: string;
  alt?: string;
  fallback?: string;
};

const meta = {
  title: "shadcn Primitives/Avatar",
  component: Avatar,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    src: {
      control: { type: "text" },
      table: { category: "Content" },
    },
    alt: {
      control: { type: "text" },
      table: { category: "Content" },
    },
    fallback: {
      control: { type: "text" },
      table: { category: "Content" },
    },
    className: {
      control: { type: "text" },
      table: { category: "Style" },
    },
  },
  args: {
    src: "https://github.com/shadcn.png",
    alt: "Colin",
    fallback: "CN",
  },
} satisfies Meta<AvatarArgs>;

export default meta;

type Story = StoryObj<AvatarArgs>;

const renderAvatar = ({ src, alt, fallback, ...props }: AvatarArgs) => (
  <Avatar {...props}>
    <AvatarImage src={src} alt={alt} />
    <AvatarFallback>{fallback}</AvatarFallback>
  </Avatar>
);

export const Default: Story = {
  render: renderAvatar,
};
