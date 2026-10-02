package autoapprove

// Change is a normalized, provider-agnostic view of a proposed change that an
// approval gate is asked to clear: a pull request, a commit, or a deploy. It is
// built from the incoming event payload on a best-effort basis. When the payload
// cannot be understood, Known is false and the change is treated as high risk.
type Change struct {
	Paths []string
	Known bool
}

// ChangeFromPayload extracts a Change from an event payload. It reads the common
// shapes emitted by version-control triggers (a list of changed files). It fails
// closed: if a files list is present but any entry cannot be resolved to a path,
// Known is false so the whole change is treated as high risk rather than being
// silently classified on a partial file list.
func ChangeFromPayload(data any) Change {
	m, ok := data.(map[string]any)
	if !ok {
		return Change{Known: false}
	}

	paths, clean := extractPaths(m)
	return Change{Paths: paths, Known: clean && len(paths) > 0}
}

func extractPaths(m map[string]any) ([]string, bool) {
	for _, key := range []string{"files", "paths", "changed_files", "files_changed"} {
		raw, ok := m[key]
		if !ok {
			continue
		}
		return toStringSlice(raw)
	}
	return nil, true
}

// toStringSlice accepts either a list of path strings or a list of objects that
// carry a path under a common key. It returns clean=false if the value is not a
// list, or if any entry cannot be resolved to a non-empty path, so a payload we
// only partially understand fails closed.
func toStringSlice(raw any) ([]string, bool) {
	list, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		p := resolvePath(item)
		if p == "" {
			return nil, false
		}
		out = append(out, p)
	}
	return out, true
}

func resolvePath(item any) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]any:
		for _, key := range []string{"path", "filename", "file"} {
			if s, ok := v[key].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
