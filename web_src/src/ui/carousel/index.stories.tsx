import type { Meta, StoryObj } from "@storybook/react";

import { Carousel, CarouselContent, CarouselItem, CarouselNext, CarouselPrevious } from "./index";

type CarouselArgs = React.ComponentProps<typeof Carousel>;

const meta = {
  title: "shadcn Primitives/Carousel",
  component: Carousel,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    orientation: {
      control: { type: "radio" },
      options: ["horizontal", "vertical"],
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    orientation: "horizontal",
    className: "w-full max-w-xs",
  },
} satisfies Meta<CarouselArgs>;

export default meta;

type Story = StoryObj<CarouselArgs>;

const slides = Array.from({ length: 5 }).map((_, index) => index + 1);

const renderCarousel = (args: CarouselArgs) => (
  <Carousel {...args}>
    <CarouselContent className="p-1">
      {slides.map((slide) => (
        <CarouselItem key={slide}>
          <div className="bg-muted flex h-40 items-center justify-center rounded-lg">
            <span className="text-lg font-semibold">Slide {slide}</span>
          </div>
        </CarouselItem>
      ))}
    </CarouselContent>
    <CarouselPrevious />
    <CarouselNext />
  </Carousel>
);

export const Default: Story = {
  render: renderCarousel,
};

export const Vertical: Story = {
  args: {
    orientation: "vertical",
    className: "h-96 w-72",
  },
  render: renderCarousel,
};
