import type { Meta, StoryObj } from "@storybook/react";
import { Http } from "./";

const meta: Meta<typeof Http> = {
  title: "ui/Http",
  component: Http,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  args: {
    title: "Make HTTP Request",
    headerColor: "bg-gray-50",
    method: "POST",
    url: "https://api.example.com/v1/users",
    payload: {
      name: "John Doe",
      email: "john@example.com",
    },
    headers: [
      { name: "Content-Type", value: "application/json" },
      { name: "Authorization", value: "Bearer token123" },
      { name: "X-Request-ID", value: "abc-123-def-456" },
    ],
    lastExecution: {
      statusCode: 200,
      receivedAt: new Date(Date.now() - 60000), // 1 minute ago
      state: "success",
    },
  },
};

export const Failed: Story = {
  args: {
    title: "Make HTTP Request",
    headerColor: "bg-gray-50",
    method: "GET",
    url: "https://api.example.com/v1/data",
    headers: [{ name: "Content-Type", value: "application/json" }],
    lastExecution: {
      statusCode: 404,
      receivedAt: new Date(Date.now() - 120000), // 2 minutes ago
      state: "failed",
    },
  },
};

export const Running: Story = {
  args: {
    title: "Make HTTP Request",
    headerColor: "bg-gray-50",
    method: "PUT",
    url: "https://api.example.com/v1/resource/123",
    payload: {
      status: "active",
    },
    headers: [{ name: "Content-Type", value: "application/json" }],
    lastExecution: {
      receivedAt: new Date(),
      state: "running",
    },
  },
};

export const NoExecution: Story = {
  args: {
    title: "Make HTTP Request",
    headerColor: "bg-gray-50",
    method: "POST",
    url: "https://api.example.com/v1/users",
  },
};

export const Collapsed: Story = {
  args: {
    title: "Make HTTP Request",
    headerColor: "bg-gray-50",
    collapsed: true,
    collapsedBackground: "bg-gray-100",
    method: "POST",
    url: "https://api.example.com/v1/users",
    lastExecution: {
      statusCode: 200,
      receivedAt: new Date(),
      state: "success",
    },
  },
};
