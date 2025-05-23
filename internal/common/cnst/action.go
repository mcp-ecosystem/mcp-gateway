package cnst

// ActionType represents the type of action performed on a configuration
type ActionType string

const (
	// ActionCreate represents a create action
	ActionCreate ActionType = "Create"
	// ActionUpdate represents an update action
	ActionUpdate ActionType = "Update"
	// ActionDelete represents a delete action
	ActionDelete ActionType = "Delete"
	// ActionRevert represents a revert action
	ActionRevert ActionType = "Revert"
)
