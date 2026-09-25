· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082624891)
<a href="#"><img alt="P2" src="https://greptile-static-assets.s3.amazonaws.com/badges/p2.svg?v=9" align="top"></a> **Parent synchronization runs during render**

`syncFromField()` copies parent field values into mutable snapshots directly during render. This violates the repository directive, “Keep local state in sync with parent state via useEffect.” Move parent-value reconciliation into an effect, while retaining the explicit synchronization needed when a focus event activates another field. This repository requirement must be satisfied before merging.

**Context Used:** web_src/AGENTS.md ([source](https://github.com/superplanehq/superplane/blob/main/web_src/AGENTS.md))

Note: If this suggestion doesn't match your team's coding style, reply to this and let me know. I'll remember it for next time!
