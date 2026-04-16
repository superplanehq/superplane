package digitalocean

type DatabaseClusterNodeMetadata struct {
	DatabaseClusterID   string `json:"databaseClusterId" mapstructure:"databaseClusterId"`
	DatabaseClusterName string `json:"databaseClusterName" mapstructure:"databaseClusterName"`
}
