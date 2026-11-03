package contracts

import "context"

// OperatorDiagnostics returns only public-safe operational state, never paths,
// SQL errors, credentials, or internal backup command output.
type OperatorDiagnostics interface {
	Status(context.Context) OperatorStatus
}
type OperatorComponent struct {
	Status       string  `json:"status"`
	DiagnosticID *string `json:"diagnosticId"`
}
type OperatorBackup struct {
	Status        string  `json:"status"`
	LastSuccessAt *string `json:"lastSuccessAt"`
	NextDueAt     *string `json:"nextDueAt"`
	DiagnosticID  *string `json:"diagnosticId"`
}
type OperatorStatus struct {
	Version     string            `json:"version"`
	StartedAt   string            `json:"startedAt"`
	Ready       bool              `json:"ready"`
	Database    OperatorComponent `json:"database"`
	Attachments OperatorComponent `json:"attachments"`
	Backup      OperatorBackup    `json:"backup"`
}
