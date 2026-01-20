# Adding New Workflow Templates

This guide explains how to create and add new workflow templates to SuperPlane.

## Table of Contents

- [Overview](#overview)
- [What are Templates?](#what-are-templates)
- [Creating a New Template](#creating-a-new-template)
- [Template File Structure](#template-file-structure)
- [Adding Templates to SuperPlane](#adding-templates-to-superplane)
- [Template Validation](#template-validation)
- [Best Practices](#best-practices)

---

## Overview

Templates provide pre-built workflow patterns that users can use as starting points when creating new canvases. They help standardize common workflows and make it easier for teams to get started with SuperPlane.

---

## What are Templates?

Templates are YAML files that define complete workflow configurations including:
- Workflow metadata (name, description)
- Nodes (triggers, components, widgets)
- Edges (connections between nodes)
- Node positioning and configuration

When users create a new canvas, they can select a template, and SuperPlane will automatically populate the canvas with the template's nodes and edges.

---

## Creating a New Template

There are two ways to create a new template:

### Method 1: Design in the UI (Recommended)

1. **Create a workflow in the SuperPlane UI**
   - Navigate to any canvas in your local development environment
   - Design your workflow using the visual canvas editor
   - Add and configure all the nodes (triggers, components) you want in the template
   - Connect the nodes with edges to define the workflow flow

2. **Export the workflow**
   - Once your workflow is complete, locate the "Export YAML" dropdown in the top-right of the canvas header
   - **Note**: This feature is only available when running SuperPlane in development mode (the `isDev` flag must be enabled)
   - Click "Export YAML" and select either:
     - "Download File" to save the YAML file directly
     - "Copy to Clipboard" to copy the YAML and manually save it

3. **Prepare the template**
   - Open the exported YAML file in your text editor
   - Ensure the `metadata.isTemplate` field is set to `true`
   - Update the name and description to be clear and helpful
   - Review and adjust node positions if needed

### Method 2: Write YAML Manually

You can also create a template by writing the YAML file from scratch. Use the existing template as a reference:
- See `/templates/canvases/policy-gated-deployment.yaml` for an example

---

## Template File Structure

A template YAML file has two main sections:

### Metadata Section

```yaml
metadata:
  name: "Template Name"
  description: "Brief description of what this template does"
  isTemplate: true
```

**Required fields:**
- `name` (string): Display name for the template
- `description` (string): Description shown to users when selecting templates
- `isTemplate` (boolean): Must be `true` to mark this as a template

### Spec Section

```yaml
spec:
  nodes:
    - id: "unique-node-id"
      name: "Display Name"
      type: "TYPE_TRIGGER" # or TYPE_COMPONENT, TYPE_WIDGET
      configuration: {}
      position:
        x: 100
        y: 200
      trigger:
        name: "start"
      # ... other node properties
  edges:
    - sourceId: "source-node-id"
      targetId: "target-node-id"
      channel: "default"
```

**Nodes** define the individual steps in the workflow:
- `id`: Unique identifier for the node (used in edges)
- `name`: Display name shown in the UI
- `type`: Node type (`TYPE_TRIGGER`, `TYPE_COMPONENT`, or `TYPE_WIDGET`)
- `configuration`: Component-specific configuration
- `position`: X and Y coordinates for canvas layout
- `trigger`, `component`, or `widget`: Type-specific configuration

**Edges** define the connections between nodes:
- `sourceId`: ID of the starting node
- `targetId`: ID of the ending node
- `channel`: Output channel name (e.g., "default", "approved", "rejected")

---

## Adding Templates to SuperPlane

1. **Save your template file**
   - Place the YAML file in `/templates/canvases/` directory
   - Use a descriptive, kebab-case filename (e.g., `my-template-name.yaml`)

2. **Restart the server**
   - Templates are automatically loaded when the server starts
   - Stop your local development server
   - Run `make dev.start` to restart

3. **Verify the template**
   - Navigate to your local SuperPlane instance (typically http://localhost:8000 when using `make dev.start`)
   - Click "Create Canvas" or similar action
   - Your template should appear in the template selection list

---

## Template Validation

The template seeding process validates:
- YAML syntax is correct
- Required metadata fields are present (`name`, `description`)
- Node and edge structures are valid

If a template fails validation:
- Check the server logs for error messages
- Common issues:
  - Missing `isTemplate: true` in metadata
  - Invalid YAML syntax
  - Missing required fields in nodes or edges

---

## Best Practices

### Template Design

1. **Keep it simple**: Templates should demonstrate clear, common patterns
2. **Use descriptive names**: Node names should clearly indicate their purpose
3. **Include helpful configurations**: Pre-configure nodes with sensible defaults
4. **Document complex workflows**: Use clear node names and descriptions

### Node Positioning

- Space nodes appropriately for readability
- Use consistent horizontal spacing between sequential nodes
- Group related nodes vertically for parallel paths
- Ensure the entire workflow fits in a reasonable viewport

### Configuration

- Use realistic but generic configuration values
- Avoid hardcoding organization-specific values
- Set component configurations that work out-of-the-box when possible
- Use placeholder values that clearly need user customization (e.g., "YOUR_API_KEY")

### Testing

Before submitting a template:
1. Create a new canvas from your template in the UI
2. Verify all nodes appear correctly positioned
3. Check that all edges are properly connected
4. Test the workflow with sample data if possible
5. Ensure the template is useful for its intended purpose

### Metadata

- **Name**: Use clear, descriptive names (e.g., "Policy-gated deployment", not "Template 1")
- **Description**: Explain when and why someone would use this template
- Make sure `isTemplate: true` is set

---

## Example: Policy-Gated Deployment Template

See `/templates/canvases/policy-gated-deployment.yaml` for a complete example that demonstrates:
- Manual trigger for deployment requests
- Policy validation with a filter component
- Time-based gating for business hours
- Approval workflow
- Conditional branching (approved vs. rejected paths)

This template can serve as a reference for creating your own templates.

---

## Questions or Issues?

If you encounter problems or have questions about templates:
- Check the server logs for error messages
- Review the existing template files for examples
- Consult the [Component Implementation Patterns](component-implementations.md) guide for component-specific details
