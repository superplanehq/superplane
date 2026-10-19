package messages

const ApplicationEncryptionMigrationRequestedRoutingKey = "application-encryption-migration-requested"

func PublishApplicationEncryptionMigrationRequested() error {
	return Publish(CanvasExchange, ApplicationEncryptionMigrationRequestedRoutingKey, nil)
}
