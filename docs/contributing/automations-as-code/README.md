# Custom automation YAML v2

Each file defines one custom automation. Select its factory and stage on the
board or supply them in the API request. Use [the schema](automation-v2.schema.json)
for editor completion.

- [Backlog example](backlog.yaml) adds manual and stage-entry triggers. Its
  steps inspect the checkout and summarize the task.
- [Schedule example](schedules.yaml) uses two schedules with distinct IDs.
- [Invalid example](invalid.yaml) uses an unsupported trigger. Validation
  reports the problem at `triggers[0].use`.

## Document fields

| Field | Rule |
| --- | --- |
| `apiVersion` | Required. Use `v2`. |
| `kind` | Required. Use `Automation`. |
| `name` | Required. Enter a nonblank display name. |
| `icon` | Optional. Use a key from the automation icon picker. The default is `sparkles`. |
| `timeout_seconds` | Optional. Enter an integer from `1` through `86400`. The default is `3600`. The timeout covers setup and all steps. |
| `agent` | Optional. The factory agent applies if you omit this field. |
| `triggers` | Required. Enter one or more triggers. Any trigger can start the automation. |
| `steps` | Required. Enter one or more steps in execution order. All triggers use these steps. |

Use one YAML mapping per file. Field names and enum values are case-sensitive.
You can use comments, quoted strings, and multiline bodies. The parser rejects
unknown fields, duplicate keys, incorrect types, null values, and YAML aliases.
Validation reports the field path, line, and column of an error.

## Triggers

| `use` | Stages | Behavior |
| --- | --- | --- |
| `manual` | All | Starts when a user runs the automation from an eligible task. |
| `task_enters_stage` | Backlog, Implement, Done | Starts when a task enters the selected stage. |
| `pull_request_updated` | Verify | Starts for the current revision of each eligible pull request when its task enters Verify. Starts again for eligible new revisions. |
| `schedule` | All | Starts on the schedule and checks out the default branch. In Backlog, a scheduled run can create a task. |

Use each event trigger once. To give a trigger a public identifier, set its
optional `id` to a unique, nonblank string. Give each schedule an ID when the
automation contains multiple schedules. Keep a schedule's ID to preserve its
identity when you reorder triggers.

Use `config` on schedule triggers. Set `config.cron` to a string of five cron
fields, or six with seconds first. Set the optional `config.utc_offset` to a
number of hours from `-12` through `14`. The default is `0`. Fractional values
such as `5.5` work. The offset stays fixed throughout the year.

## Steps

| Field | Rule |
| --- | --- |
| `name` | Required. Enter a nonblank label. Labels can repeat. |
| `bash` | Enter a nonblank Bash script if the step runs a command. |
| `prompt` | Enter nonblank instructions if the step uses the agent. |
| `working_directory` | Optional. Enter an absolute or relative path on one line. A relative path must stay within the runner's working directory. Omit this field to use the runner's working directory. |
| `secrets` | Optional for Bash. Use a list of distinct, nonblank secret names from the organization. |

Set exactly one of `bash` and `prompt` in each step. To keep line breaks in a
script or prompt, use a YAML literal block with `|`. Use `|-` to remove its
final line break.

The runner grants access to all referenced secrets for the whole runner task.
Each step can access the secrets from that task. Reference secrets by their
organization names and keep their values out of YAML.

## Agent

| Field | Rule |
| --- | --- |
| `harness` | Required in `agent`. Use `superplane`, `claude-code`, `codex`, or `openrouter`. |
| `model` | Optional. Use an available model ID for the selected configuration. The applicable default model applies if you omit this field. |
| `credentials` | Optional. The factory configuration applies if you omit this field. Omit this field with `superplane` to use the workspace configuration. |
| `credentials.source` | Required in `credentials`. Use `integration` or `subscription`. |
| `credentials.name` | Required with `integration`. Enter the name of a ready, compatible integration in the organization. |

To use a configured ChatGPT subscription, set `harness` to `codex` and set
`credentials.source` to `subscription`. Omit `credentials.name` and `model`.

## Try the prototype locally

1. Start SuperPlane with `make dev.up`, `make dev.setup`, and `make dev.server`.
2. Open the organization in Installation Admin. In Experimental Features, enable
   **Factories**, **Custom Automations**, and **Automations as code**.
3. Open a factory with a connected repository and a configured local runner.
4. Create an automation in Backlog. Add a manual trigger and a Bash step.
5. Switch to **YAML**, paste the definition below, and change the marker.
6. Switch to **Canvas** to confirm the step changed. Save the automation.
7. Run the automation from a Backlog task. Open **Runs** and find the marker.
8. Change the marker with the HTTP update below. Run it again and check the new log.
9. Open the automation under **Automations** to inspect its generated graph.
   The inspector uses the existing canvas and run permissions.

```yaml
# Save as /tmp/backlog-marker.yaml to use the HTTP examples.
apiVersion: v2
kind: Automation
name: Print a Backlog marker
triggers:
  - use: manual
steps:
  - name: Print the marker
    bash: printf '%s\n' automation-yaml-v2
```

On a legacy automation, the first YAML read shows a definition from the live
graph. Save it to keep its ID and run history and store its source. A visual edit
can normalize the YAML. Switching views alone preserves the original source.
If two editors save, the last successful save becomes the current revision.

### HTTP

Create an organization API key with the Maintainer or Admin role. Leave its
canvas scope empty. Set `FACTORY_ID` to the factory UUID. The commands use Bash,
`curl`, and `jq`.

```bash
BASE=http://localhost:8000
FACTORY_ID=00000000-0000-0000-0000-000000000000
read -rsp 'Organization API key: ' TOKEN
printf '\n'

API="$BASE/api/v2/automations"
auth=(-H "Authorization: Bearer $TOKEN")
json=(-H 'Content-Type: application/json')

jq -n --arg factory_id "$FACTORY_ID" --arg stage backlog \
  --rawfile yaml /tmp/backlog-marker.yaml '{factory_id:$factory_id, stage:$stage, yaml:$yaml}' \
  >/tmp/automation-create.json

curl -fsS "${auth[@]}" "${json[@]}" --data-binary @/tmp/automation-create.json "$API:validate"

AUTOMATION_ID=$(curl -fsS "${auth[@]}" "${json[@]}" \
  --data-binary @/tmp/automation-create.json "$API" | jq -er .id)

curl -fsS "${auth[@]}" "$API?factory_id=$FACTORY_ID"
curl -fsS "${auth[@]}" "$API/$AUTOMATION_ID" | jq -rj .yaml >/tmp/backlog-marker.yaml

# Edit /tmp/backlog-marker.yaml, then save the full source.
jq -n --rawfile yaml /tmp/backlog-marker.yaml '{yaml:$yaml}' \
  | curl -fsS -X PUT "${auth[@]}" "${json[@]}" --data-binary @- "$API/$AUTOMATION_ID"

# Remove the example automation when finished.
curl -fsS -X DELETE "${auth[@]}" "$API/$AUTOMATION_ID"
```
