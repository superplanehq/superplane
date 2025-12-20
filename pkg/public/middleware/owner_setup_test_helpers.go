package middleware

// ResetOwnerSetupStateForTests re-enables the owner setup flow and clears
// the cached state so tests can control whether setup is required.
func ResetOwnerSetupStateForTests() {
	ownerSetupEnabled = true

	ownerSetupMu.Lock()
	ownerSetupNeededCache = nil
	ownerSetupMu.Unlock()
}
