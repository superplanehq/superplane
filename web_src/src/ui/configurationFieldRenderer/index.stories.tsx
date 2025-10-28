import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { ConfigurationFieldRenderer } from './index';
import { ComponentsConfigurationField } from '../../api-client';
import { TooltipProvider } from '../tooltip';
import React from 'react';

const meta: Meta<typeof ConfigurationFieldRenderer> = {
  title: 'ui/ConfigurationFieldRenderer',
  component: ConfigurationFieldRenderer,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
  decorators: [
    (Story) => (
      <TooltipProvider>
        <div className="max-w-2xl">
          <Story />
        </div>
      </TooltipProvider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof ConfigurationFieldRenderer>;

const Wrapper = ({ field, initialValue }: { field: ComponentsConfigurationField; initialValue?: any }) => {
  const [value, setValue] = useState(initialValue);
  return (
    <div className="space-y-4">
      <ConfigurationFieldRenderer
        field={field}
        value={value}
        onChange={setValue}
      />
      <div className="mt-4 p-4 bg-gray-100 dark:bg-zinc-800 rounded-md">
        <p className="text-xs font-mono">Current value:</p>
        <pre className="text-xs">{JSON.stringify(value, null, 2)}</pre>
      </div>
    </div>
  );
};

export const StringField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'username',
        label: 'Username',
        type: 'string',
        description: 'Enter your username',
        required: true,
      }}
      initialValue=""
    />
  ),
};

export const NumberField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'age',
        label: 'Age',
        type: 'number',
        description: 'Enter your age',
        typeOptions: {
          number: {
            min: 0,
            max: 120,
          },
        },
      }}
      initialValue={25}
    />
  ),
};

export const BooleanField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'enabled',
        label: 'Enable Feature',
        type: 'boolean',
        description: 'Toggle to enable or disable this feature',
      }}
      initialValue={false}
    />
  ),
};

export const SelectField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'priority',
        label: 'Priority',
        type: 'select',
        description: 'Select the priority level',
        typeOptions: {
          select: {
            options: [
              { label: 'Low', value: 'low' },
              { label: 'Medium', value: 'medium' },
              { label: 'High', value: 'high' },
              { label: 'Critical', value: 'critical' },
            ],
          },
        },
      }}
      initialValue="medium"
    />
  ),
};

export const MultiSelectField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'tags',
        label: 'Tags',
        type: 'multi-select',
        description: 'Select multiple tags',
        typeOptions: {
          multiSelect: {
            options: [
              { label: 'Bug', value: 'bug' },
              { label: 'Feature', value: 'feature' },
              { label: 'Documentation', value: 'docs' },
              { label: 'Testing', value: 'testing' },
            ],
          },
        },
      }}
      initialValue={['bug', 'feature']}
    />
  ),
};

export const DateField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'dueDate',
        label: 'Due Date',
        type: 'date',
        description: 'Select a due date',
      }}
      initialValue={new Date().toISOString()}
    />
  ),
};

export const UrlField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'website',
        label: 'Website URL',
        type: 'url',
        description: 'Enter a valid URL',
      }}
      initialValue="https://example.com"
    />
  ),
};

export const TimeField: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'meetingTime',
        label: 'Meeting Time',
        type: 'time',
        description: 'Select a meeting time',
      }}
      initialValue="14:30"
    />
  ),
};

export const ListFieldSimple: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'emails',
        label: 'Email Addresses',
        type: 'list',
        description: 'Add multiple email addresses',
        typeOptions: {
          list: {
            itemDefinition: {
              type: 'string',
            },
          },
        },
      }}
      initialValue={['user@example.com', 'admin@example.com']}
    />
  ),
};

export const ListFieldWithObjects: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'contacts',
        label: 'Contacts',
        type: 'list',
        description: 'Add contact information',
        typeOptions: {
          list: {
            itemDefinition: {
              type: 'object',
              schema: [
                {
                  name: 'name',
                  label: 'Name',
                  type: 'string',
                  required: true,
                },
                {
                  name: 'email',
                  label: 'Email',
                  type: 'string',
                },
                {
                  name: 'role',
                  label: 'Role',
                  type: 'select',
                  typeOptions: {
                    select: {
                      options: [
                        { label: 'Developer', value: 'dev' },
                        { label: 'Designer', value: 'designer' },
                        { label: 'Manager', value: 'manager' },
                      ],
                    },
                  },
                },
              ],
            },
          },
        },
      }}
      initialValue={[
        { name: 'John Doe', email: 'john@example.com', role: 'dev' },
        { name: 'Jane Smith', email: 'jane@example.com', role: 'designer' },
      ]}
    />
  ),
};

export const ObjectFieldWithSchema: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'address',
        label: 'Address',
        type: 'object',
        description: 'Enter address information',
        typeOptions: {
          object: {
            schema: [
              {
                name: 'street',
                label: 'Street',
                type: 'string',
                required: true,
              },
              {
                name: 'city',
                label: 'City',
                type: 'string',
                required: true,
              },
              {
                name: 'state',
                label: 'State',
                type: 'string',
              },
              {
                name: 'zipCode',
                label: 'Zip Code',
                type: 'string',
              },
              {
                name: 'isPrimary',
                label: 'Primary Address',
                type: 'boolean',
              },
            ],
          },
        },
      }}
      initialValue={{
        street: '123 Main St',
        city: 'San Francisco',
        state: 'CA',
        zipCode: '94102',
        isPrimary: true,
      }}
    />
  ),
};

export const ObjectFieldWithJSON: Story = {
  render: () => (
    <Wrapper
      field={{
        name: 'metadata',
        label: 'Metadata',
        type: 'object',
        description: 'Enter custom metadata as JSON',
      }}
      initialValue={{
        version: '1.0',
        author: 'John Doe',
        tags: ['important', 'draft'],
      }}
    />
  ),
};

export const ComplexForm: Story = {
  render: () => {
    const [values, setValues] = useState<Record<string, any>>({
      projectName: 'My Project',
      description: 'A sample project',
      priority: 'high',
      enabled: true,
      teamMembers: [
        { name: 'Alice', role: 'lead' },
        { name: 'Bob', role: 'developer' },
      ],
    });

    const fields: ComponentsConfigurationField[] = [
      {
        name: 'projectName',
        label: 'Project Name',
        type: 'string',
        required: true,
        description: 'Enter the project name',
      },
      {
        name: 'description',
        label: 'Description',
        type: 'string',
        description: 'Describe your project',
      },
      {
        name: 'priority',
        label: 'Priority',
        type: 'select',
        typeOptions: {
          select: {
            options: [
              { label: 'Low', value: 'low' },
              { label: 'Medium', value: 'medium' },
              { label: 'High', value: 'high' },
            ],
          },
        },
      },
      {
        name: 'enabled',
        label: 'Active',
        type: 'boolean',
      },
      {
        name: 'teamMembers',
        label: 'Team Members',
        type: 'list',
        typeOptions: {
          list: {
            itemDefinition: {
              type: 'object',
              schema: [
                {
                  name: 'name',
                  label: 'Name',
                  type: 'string',
                },
                {
                  name: 'role',
                  label: 'Role',
                  type: 'select',
                  typeOptions: {
                    select: {
                      options: [
                        { label: 'Lead', value: 'lead' },
                        { label: 'Developer', value: 'developer' },
                        { label: 'Designer', value: 'designer' },
                      ],
                    },
                  },
                },
              ],
            },
          },
        },
      },
    ];

    return (
      <div className="space-y-6">
        {fields.map((field) => (
          <ConfigurationFieldRenderer
            key={field.name}
            field={field}
            value={values[field.name!]}
            onChange={(value) => setValues({ ...values, [field.name!]: value })}
            allValues={values}
          />
        ))}
        <div className="mt-4 p-4 bg-gray-100 dark:bg-zinc-800 rounded-md">
          <p className="text-xs font-mono mb-2">Form Values:</p>
          <pre className="text-xs">{JSON.stringify(values, null, 2)}</pre>
        </div>
      </div>
    );
  },
};
