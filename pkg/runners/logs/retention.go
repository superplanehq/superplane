package logs

import "bytes"

// RetainContent applies the installation-wide active-log size limit.
// Stores call it while holding their append lock so totalBytes remains stable.
func RetainContent(totalBytes int64, content []byte) ([]byte, bool) {
	contentLimit := MaxRetainedBytes - int64(len(TruncationRecord))
	if totalBytes+int64(len(content)) < contentLimit {
		return append([]byte(nil), content...), false
	}

	available := contentLimit - totalBytes
	if available < 0 {
		available = 0
	}
	retained := completeRecordsWithin(content, available)
	retained = append(retained, TruncationRecord...)
	return retained, true
}

func completeRecordsWithin(content []byte, limit int64) []byte {
	var retained []byte
	for len(content) > 0 {
		newline := bytes.IndexByte(content, '\n')
		if newline < 0 {
			break
		}
		recordBytes := int64(newline + 1)
		if int64(len(retained))+recordBytes > limit {
			break
		}
		retained = append(retained, content[:newline+1]...)
		content = content[newline+1:]
	}
	return retained
}
