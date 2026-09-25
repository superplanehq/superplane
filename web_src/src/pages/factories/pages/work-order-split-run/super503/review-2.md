· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082286823)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Returning focus duplicates spoken words**

Switching away from a field and back commits its provisional words too early. For example, let description `"Notes"` receive interim `"hello"`, focus the title, then return to the description before the final result. `resetSnapshot()` adopts `"Notes hello"` as committed text and clears the live phrase. Final `"hello"` then produces `"Notes hello hello"` instead of replacing the provisional words.

Keep the committed snapshot and live phrase per field, or end the pending recognition segment when switching fields. Add a regression test that switches back before finalization.
