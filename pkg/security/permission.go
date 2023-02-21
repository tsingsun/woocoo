package security

// PermissionItem is for describing a permission.
type PermissionItem struct {
	// AppCode is the application code which the action belongs to.
	AppCode string
	// Action name
	Action string
	// Operator name
	Operator string
}
