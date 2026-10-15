# System skill updates

Define supplied skill templates in `pkg/components/runner/system_skill_templates.go`.
Keep each template ID stable. Increment its version when instructions or frontmatter change.

The system skill sync worker runs at startup and once per hour. It scans active
workspaces across organizations where skills are enabled. It updates a skill only
when its content hash matches its stored baseline. Customized skills and unknown
templates remain unchanged. The worker updates instructions, version, and baseline
without changing the skill ID, command name, or enabled state.

Each write compares the saved configuration with the scanned configuration. An
edit or restore after the scan prevents that write. Workers from an older release
also skip skills whose stored template version is newer than their own template.

Set `START_SYSTEM_SKILL_SYNC_WORKER=yes` on the worker process. Development Compose,
the Helm worker deployment, the demo image, and new single-host installations enable it. Existing
single-host installations must add this flag to `superplane.env` and restart.

This worker updates existing system skills. It does not install missing skills or
change automation prompts. Installation and default automation changes are separate.
