· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082395369)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Returning focus discards retained words**

Let description `"Notes"` receive interim `"hello"`, switch to the title, and let `"hello"` finalize there. Return to the description and dictate a new phrase, `"next"`. The restored snapshot still treats `"hello"` as provisional, so the next result produces `"Notes next"` instead of `"Notes hello next"`, discarding the retained words.

Final callbacks update only the active field, so inactive snapshots never learn that the previous recognition segment ended. Track segment completion across snapshots and preserve retained words when returning after finalization. Replacement should apply only when returning during the same pending segment.

Note: If this suggestion doesn't match your team's coding style, reply to this and let me know. I'll remember it for next time!

· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082395381)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Cached snapshots overwrite updated titles**

In the create-task request dialog, start dictation, focus the title, and return to the description before speaking. Dictating into the description updates the automatically derived title, but its cached snapshot still contains the old title. Refocusing the title restores that old prefix, so the next recognition result overwrites the updated title and marks it as manually edited.

Focus only mutates refs, so the render-time reconciliation does not run before that result. Reconcile the saved snapshot against `getValue()` inside `activateField` before accepting more speech.
