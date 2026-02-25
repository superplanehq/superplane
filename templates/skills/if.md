# If Component Skill

Use this guidance when planning or configuring the `if` component.

## Purpose

The `if` component evaluates a boolean expression and routes execution to one of two output channels:

- `true`
- `false`

## Required Configuration

- `expression` (required): an expression that must evaluate to a boolean.

If the expression does not compile or does not return a boolean, execution fails.

## Expression Context

Expressions can reference:

- `$` for run context data
- `root()` for root event data
- `previous()` or `previous(<depth>)` for prior node outputs by depth

## Planning Rules

When generating workflow operations that include `if`:

1. Always set `configuration.expression`.
2. Use clear boolean comparisons (for example equality, numeric thresholds, or existence checks).
3. After adding `if`, connect downstream steps from the correct channel:
   - happy path / matched condition from `true`
   - fallback / unmatched condition from `false`
4. Do not invent additional output channels for this component.

## Good Expression Examples

- `$["Approval"].status == "approved"`
- `$["RiskScore"].value >= 80`
- `previous().status == "success"`

## Common Mistakes To Avoid

- Returning a non-boolean expression (for example raw strings or objects).
- Connecting branches without specifying the proper `true` or `false` source channel.
- Leaving `expression` empty.
