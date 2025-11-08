import { Fragment } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ChevronRight, Loader2, Mail, Settings, Trash2 } from "lucide-react";

import { Button, type ButtonProps } from "./index";

type IconVariant = "none" | "withIcon" | "iconOnly";
type ButtonStoryArgs = ButtonProps & { iconVariant?: IconVariant };

const meta = {
  title: "shadcn Primitives/Button",
  component: Button,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: { type: "select" },
      options: ["default", "destructive", "outline", "secondary", "ghost", "link"],
      table: { category: "Style" },
    },
    size: {
      control: { type: "select" },
      options: ["default", "sm", "lg"],
      table: { category: "Layout" },
    },
    disabled: {
      control: { type: "boolean" },
      table: { category: "State" },
    },
    children: {
      control: { type: "text" },
      table: { category: "Content" },
    },
    iconVariant: {
      control: { type: "select" },
      options: ["none", "withIcon", "iconOnly"],
      table: { category: "Layout" },
    },
  },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<ButtonStoryArgs>;

const renderButton = ({ iconVariant = "none", children, size, ...props }: ButtonStoryArgs) => {
  const content = (() => {
    switch (iconVariant) {
      case "withIcon":
        return (
          <>
            <Mail className="mr-2 h-4 w-4" />
            {children}
          </>
        );
      case "iconOnly":
        return <Settings className="h-4 w-4" />;
      default:
        return children;
    }
  })();

  return (
    <Button {...props} size={iconVariant === "iconOnly" ? "icon" : size}>
      {content}
    </Button>
  );
};

const baseArgs: ButtonStoryArgs = {
  variant: "default",
  size: "default",
  disabled: false,
  children: "Button",
  iconVariant: "none",
};

const iconVariantOrder: IconVariant[] = ["none", "withIcon", "iconOnly"];
const iconVariantLabels: Record<IconVariant, string> = {
  none: "No icon",
  withIcon: "Leading icon",
  iconOnly: "Icon only",
};

const sizeOrder: Array<NonNullable<ButtonProps["size"]>> = ["sm", "default", "lg"];
const sizeLabels: Record<NonNullable<ButtonProps["size"]>, string> = {
  sm: "Small",
  default: "Default",
  lg: "Large",
  icon: "Icon",
};

export const Default: Story = {
  args: baseArgs,
  render: renderButton,
};

export const WithIcon: Story = {
  args: {
    ...baseArgs,
    children: "Email",
    iconVariant: "withIcon",
  },
  render: renderButton,
  argTypes: {
    iconVariant: { control: { disable: true } },
  },
};

export const WithTrailingIcon: Story = {
  args: {
    ...baseArgs,
    children: (
      <>
        Next
        <ChevronRight className="ml-2 h-4 w-4" />
      </>
    ),
  },
  argTypes: {
    children: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const IconButton: Story = {
  args: {
    ...baseArgs,
    variant: "outline",
    iconVariant: "iconOnly",
  },
  render: (args) => (
    <Button {...args} size="icon">
      <ChevronRight className="h-4 w-4" />
    </Button>
  ),
  argTypes: {
    children: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
    size: { control: { disable: true } },
  },
};

export const Loading: Story = {
  args: {
    ...baseArgs,
    disabled: true,
    children: (
      <>
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        Please wait
      </>
    ),
  },
  argTypes: {
    children: { control: { disable: true } },
    disabled: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const Destructive: Story = {
  args: {
    ...baseArgs,
    variant: "destructive",
    children: (
      <>
        <Trash2 className="mr-2 h-4 w-4" />
        Delete
      </>
    ),
  },
  argTypes: {
    children: { control: { disable: true } },
    variant: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const Secondary: Story = {
  args: {
    ...baseArgs,
    variant: "secondary",
    children: "Secondary",
  },
  render: renderButton,
  argTypes: {
    variant: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const Ghost: Story = {
  args: {
    ...baseArgs,
    variant: "ghost",
    children: "Ghost",
  },
  render: renderButton,
  argTypes: {
    variant: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const Link: Story = {
  args: {
    ...baseArgs,
    variant: "link",
    children: "Button link",
  },
  render: renderButton,
  argTypes: {
    variant: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const ButtonGroup: Story = {
  render: () => (
    <div className="flex gap-2">
      <Button>Save changes</Button>
      <Button variant="outline">Cancel</Button>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Pair primary and secondary buttons to create an action group.",
      },
    },
  },
  argTypes: {
    children: { control: { disable: true } },
    variant: { control: { disable: true } },
    size: { control: { disable: true } },
    disabled: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const FullWidth: Story = {
  render: () => <Button className="w-full">Continue</Button>,
  parameters: {
    layout: "padded",
  },
  argTypes: {
    children: { control: { disable: true } },
    variant: { control: { disable: true } },
    size: { control: { disable: true } },
    disabled: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Button size="sm">Small</Button>
      <Button>Default</Button>
      <Button size="lg">Large</Button>
    </div>
  ),
  argTypes: {
    children: { control: { disable: true } },
    size: { control: { disable: true } },
    disabled: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
  },
};

export const SizeIconMatrix: Story = {
  render: () => (
    <div className="inline-grid items-center gap-3" style={{ gridTemplateColumns: "auto repeat(3, minmax(0, 1fr))" }}>
      <div />
      {sizeOrder.map((size) => (
        <div key={`heading-${size}`} className="text-xs font-medium uppercase text-muted-foreground">
          {sizeLabels[size]}
        </div>
      ))}

      {iconVariantOrder.map((iconVariant) => (
        <Fragment key={iconVariant}>
          <div className="text-sm font-medium text-muted-foreground">{iconVariantLabels[iconVariant]}</div>
          {sizeOrder.map((size) => {
            const content = (() => {
              switch (iconVariant) {
                case "withIcon":
                  return (
                    <>
                      <Mail className="mr-2 h-4 w-4" />
                      {sizeLabels[size]}
                    </>
                  );
                case "iconOnly":
                  return <Settings className="h-4 w-4" />;
                default:
                  return sizeLabels[size];
              }
            })();

            const ariaLabel =
              iconVariant === "iconOnly" ? `${iconVariantLabels[iconVariant]} ${sizeLabels[size]}` : undefined;

            return (
              <Button
                key={`${iconVariant}-${size}`}
                size={iconVariant === "iconOnly" ? "icon" : size}
                aria-label={ariaLabel}
              >
                {content}
              </Button>
            );
          })}
        </Fragment>
      ))}
    </div>
  ),
  argTypes: {
    children: { control: { disable: true } },
    size: { control: { disable: true } },
    disabled: { control: { disable: true } },
    iconVariant: { control: { disable: true } },
    variant: { control: { disable: true } },
  },
  parameters: {
    docs: {
      description: {
        story: "Visual comparison of every size and icon combination in a single grid.",
      },
    },
  },
};
