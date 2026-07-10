import type { Meta, StoryObj } from "@storybook/react";
import { Timestamp } from "./Timestamp";
import { TimestampDetails } from "./TimestampDetails";

const meta: Meta<typeof Timestamp> = {
  title: "Components/Timestamp",
  component: Timestamp,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    display: {
      control: "radio",
      options: ["absolute", "datetime", "date", "relative"],
    },
    relativeStyle: {
      control: "radio",
      options: ["full", "abbreviated"],
    },
    includeAgo: {
      control: "boolean",
    },
    withHint: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const oneDayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();

export const Absolute: Story = {
  args: {
    date: oneDayAgo,
    display: "absolute",
  },
};

export const Datetime: Story = {
  args: {
    date: oneDayAgo,
    display: "datetime",
  },
};

export const DateOnly: Story = {
  args: {
    date: oneDayAgo,
    display: "date",
  },
};

export const Relative: Story = {
  args: {
    date: oneDayAgo,
    display: "relative",
  },
};

const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();

export const RelativeAbbreviated: Story = {
  args: {
    date: fiveMinutesAgo,
    display: "relative",
    relativeStyle: "abbreviated",
  },
};

export const RelativeAbbreviatedNoAgo: Story = {
  args: {
    date: fiveMinutesAgo,
    display: "relative",
    relativeStyle: "abbreviated",
    includeAgo: false,
  },
};

/**
 * The raw hover-card grid used inside every `Timestamp` — exported as
 * `TimestampDetails` so surfaces without a HoverCard (like Recharts tooltips)
 * can render the same Local / UTC / Relative / ISO block.
 */
export const Details: Story = {
  render: () => (
    <div className="rounded-md border border-gray-200 p-3 dark:border-gray-700">
      <TimestampDetails date={fiveMinutesAgo} />
    </div>
  ),
};

export const InTableCell: Story = {
  render: () => (
    <table className="text-sm">
      <tbody>
        <tr>
          <td className="pr-6 text-gray-500">Modified</td>
          <td>
            <Timestamp date={oneDayAgo} />
          </td>
        </tr>
        <tr>
          <td className="pr-6 text-gray-500">Created</td>
          <td>
            <Timestamp date={new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString()} />
          </td>
        </tr>
        <tr>
          <td className="pr-6 text-gray-500">Started</td>
          <td>
            <Timestamp date={fiveMinutesAgo} display="relative" relativeStyle="abbreviated" includeAgo={false} />
          </td>
        </tr>
        <tr>
          <td className="pr-6 text-gray-500">Day</td>
          <td>
            <Timestamp date={oneDayAgo} display="date" />
          </td>
        </tr>
      </tbody>
    </table>
  ),
};
