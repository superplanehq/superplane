package runners

// TaskLogToFleetLog converts a stored task log sink to the fleet API shape.
func TaskLogToFleetLog(s *TaskLogSink) *FleetTaskLog {
	if s == nil || s.Type == "" {
		return nil
	}
	ft := &FleetTaskLog{Type: s.Type}
	if s.CloudWatch != nil {
		ft.CloudWatch = &struct {
			LogGroupName  string `json:"log_group_name"`
			LogStreamName string `json:"log_stream_name"`
			Region        string `json:"region,omitempty"`
		}{
			LogGroupName:  s.CloudWatch.LogGroupName,
			LogStreamName: s.CloudWatch.LogStreamName,
			Region:        s.CloudWatch.Region,
		}
	}
	return ft
}
