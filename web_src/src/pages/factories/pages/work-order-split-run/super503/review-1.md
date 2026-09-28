· [web_src/src/pages/factories/useWorkOrderFieldDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082177406)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Field switching overwrites existing text**

Focusing the title while dictating into the description updates only `lastFieldRef`; it does not refresh the shared hook's `committedRef`. The next transcript is therefore appended to the description snapshot, but `setValue` sends it to `onTitleChange`, replacing the existing title. Previously, each final callback read the selected field's current value. Synchronize the snapshot, live phrase, and length limit when the target changes, or maintain separate dictation state per field.

· [web_src/src/lib/appendSpokenPhrase.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082177416)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Multiline drafts duplicate spoken words**

`appendSpokenPhrase` appends directly after any trailing whitespace, but this function recognizes only an ASCII-space separator. A chat draft ending in `"Notes\n"` receives interim `"hello"` and renders as `"Notes\nhello"`. On the next render, stripping fails and the entire value becomes committed, so final `"hello"` produces `"Notes\nhello hello"`. Successive interim updates also accumulate instead of replacing each other. Handle all separators supported by appending, preserve the original prefix, and add a controlled-input regression test with a render between recognition events.

· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082177425)
<a href="#"><img alt="P2" src="https://greptile-static-assets.s3.amazonaws.com/badges/p2.svg?v=9" align="top"></a> **Clipped interim text blocks corrections**

When an interim transcript reaches the field limit, `livePhraseRef` stores the full phrase but the field contains a truncated value. On the next render, stripping the full phrase fails and `committedRef` becomes the already-full field. Later corrections are appended beyond the limit and discarded. For example, a 250-character title followed by interim `"refunds"` retains `" refu"` even if the final phrase is `"returns"`. This leaves provisional text for the user to correct manually. Preserve the pre-interim snapshot or track the actual inserted range so final recognition can replace clipped text.
