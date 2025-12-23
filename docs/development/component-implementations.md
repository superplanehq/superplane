# Component Implementation Patterns

This document outlines best practices and patterns for implementing components in Superplane.

## Configuration Spec Struct

### Use Strongly Typed Spec Structs

All configuration fields defined in `Configuration()` should be included in the component's spec struct with proper types. The spec is parsed using `mapstructure.Decode()`.

**❌ Bad - Using any or map[string]any:**
```go
func (e *HTTP) Execute(ctx core.ExecutionContext) error {
    // Don't do this - requires type assertions everywhere
    payload := ctx.Configuration["payload"]
    if payload != nil {
        data, ok := payload.(map[string]any)
        // More type assertions...
    }
}
```

**✅ Good - Strongly typed spec:**
```go
type Spec struct {
    Method      string      `json:"method"`
    URL         string      `json:"url"`
    SendHeaders bool        `json:"sendHeaders"`
    Headers     []Header    `json:"headers"`
    SendBody    bool        `json:"sendBody"`
    ContentType string      `json:"contentType"`
    JSON        *any        `json:"json,omitempty"`
    XML         *string     `json:"xml,omitempty"`
    Text        *string     `json:"text,omitempty"`
    FormData    *[]KeyValue `json:"formData,omitempty"`
}

func (e *HTTP) Execute(ctx core.ExecutionContext) error {
    spec := Spec{}
    err := mapstructure.Decode(ctx.Configuration, &spec)
    if err != nil {
        return err
    }

    // Now use spec.Method, spec.URL, etc. with type safety
}
```

### Handling Conditionally Visible Fields

Fields that are conditionally visible (through `VisibilityConditions`) should be defined as **pointers with `omitempty` tags**.

**Why pointers?**
- They can be `nil` when the field is not visible/provided
- Allows distinguishing between "not set" and "set to zero value"
- Prevents errors when the user hasn't filled out conditional fields

**Example:**
```go
type Spec struct {
    SendBody    bool        `json:"sendBody"`
    ContentType string      `json:"contentType"`

    // These are only visible when SendBody is true
    JSON     *any        `json:"json,omitempty"`
    XML      *string     `json:"xml,omitempty"`
    Text     *string     `json:"text,omitempty"`
    FormData *[]KeyValue `json:"formData,omitempty"`
}

// Usage
if spec.SendBody && spec.JSON != nil {
    // Safe to use *spec.JSON
}
```

## Field Types and Renderers

### Use Semantic Field Types

Field types should be semantic and describe the **content** of the field, not just its technical format. Do not rely on field names to determine rendering behavior.

**❌ Bad - Using field names to determine behavior:**
```tsx
// In StringFieldRenderer
const isMultilineField = field.name === "payloadText" || field.name === "payloadXML";
const language = field.name === "payloadXML" ? "xml" : "plaintext";
```

This creates tight coupling between:
- Component implementation (field names)
- UI rendering logic (field name checks)

**✅ Good - Using semantic field types:**
```go
// Backend: pkg/configuration/field.go
const (
    FieldTypeString = "string"  // Single-line text input
    FieldTypeText   = "text"    // Multi-line plain text editor
    FieldTypeXML    = "xml"     // Multi-line XML editor with validation
)

// Component definition
{
    Name:  "text",
    Type:  configuration.FieldTypeText,
    Label: "Text Payload",
}
```

```tsx
// Frontend: Renderer selection by type
switch (field.type) {
    case "string":
        return <StringFieldRenderer {...commonProps} />;
    case "text":
        return <TextFieldRenderer {...commonProps} />;
    case "xml":
        return <XMLFieldRenderer {...commonProps} />;
}
```

### Renderer Responsibilities

Each field renderer should:
1. **Determine behavior based on `field.type`**, not `field.name`
2. **Be independent and reusable** across different components
3. **Handle its own validation** when type-specific (e.g., XML validation)
4. **Provide appropriate UX** for the data type (Monaco editor for text/xml, simple input for strings)

## Field Naming Conventions

### Backend Field Names

Field names in the spec struct and `Configuration()` should:
- Match the JSON field name (using json tags)
- Be concise and descriptive
- Follow Go naming conventions (PascalCase for struct fields)

**Example:**
```go
type Spec struct {
    Method   string  `json:"method"`      // Not "httpMethod"
    URL      string  `json:"url"`         // Not "endpoint" or "uri"
    XML      *string `json:"xml,omitempty"` // Not "xmlPayload" or "payloadXML"
}
```

## Validation

### Backend Validation

Implement validation in the `Setup()` method:

```go
func (e *HTTP) Setup(ctx core.SetupContext) error {
    spec := Spec{}
    err := mapstructure.Decode(ctx.Configuration, &spec)
    if err != nil {
        return err
    }

    if spec.URL == "" {
        return fmt.Errorf("url is required")
    }

    return nil
}
```

### Frontend Validation

Type-specific validation (like XML format validation) should be handled in the **field renderer**, not in generic validation logic.

```tsx
// XMLFieldRenderer.tsx
const validateXML = (xmlString: string): boolean => {
    if (!xmlString.trim()) {
        setValidationError(null);
        return true;
    }

    try {
        const parser = new DOMParser();
        const xmlDoc = parser.parseFromString(xmlString, "text/xml");
        const parseError = xmlDoc.querySelector("parsererror");

        if (parseError) {
            setValidationError("Invalid XML format");
            return false;
        }

        setValidationError(null);
        return true;
    } catch (error) {
        setValidationError("Invalid XML format");
        return false;
    }
};
```

## Summary Checklist

When implementing a new component:

- Define a strongly-typed spec struct with all configuration fields
- Use pointers with `omitempty` for conditionally visible fields
- Choose semantic field types that match the content (not just "string" for everything)
- Use appropriate field renderers based on type, not field name
- Implement validation in `Setup()` method
- The Setup() and Execute() methods should always have unit tests written for them