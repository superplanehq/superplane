import type { Meta, StoryObj } from "@storybook/react";
import { Timestamp } from "./Timestamp";

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
      options: ["absolute", "relative"],
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

export const Relative: Story = {
  args: {
    date: oneDayAgo,
    display: "relative",
  },
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
      </tbody>
    </table>
  ),
};
