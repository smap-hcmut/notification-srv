package transform

// ProjectStatus represents the current status of a project
type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "INITIALIZING" // Project is initializing
	ProjectStatusProcessing   ProjectStatus = "PROCESSING"   // Project is being processed (crawling and analysis)
	ProjectStatusDone         ProjectStatus = "DONE"         // Project finished (new phase-based status)
	ProjectStatusCompleted    ProjectStatus = "COMPLETED"    // Project finished successfully (legacy)
	ProjectStatusFailed       ProjectStatus = "FAILED"       // Project encountered fatal error
	ProjectStatusPaused       ProjectStatus = "PAUSED"       // Project temporarily stopped
)

// PhaseBasedProjectStatus represents status values for phase-based progress messages
type PhaseBasedProjectStatus string

const (
	PhaseBasedStatusInitializing PhaseBasedProjectStatus = "INITIALIZING"
	PhaseBasedStatusProcessing   PhaseBasedProjectStatus = "PROCESSING"
	PhaseBasedStatusDone         PhaseBasedProjectStatus = "DONE"
	PhaseBasedStatusFailed       PhaseBasedProjectStatus = "FAILED"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusProcessing JobStatus = "PROCESSING" // Job is actively crawling/processing
	JobStatusCompleted  JobStatus = "COMPLETED"  // Job finished all batches
	JobStatusFailed     JobStatus = "FAILED"     // Job encountered fatal error
	JobStatusPaused     JobStatus = "PAUSED"     // Job temporarily stopped
)

// Platform represents social media platforms
type Platform string

const (
	PlatformTikTok    Platform = "TIKTOK"    // TikTok platform
	PlatformYouTube   Platform = "YOUTUBE"   // YouTube platform
	PlatformInstagram Platform = "INSTAGRAM" // Instagram platform
)

// IsValidProjectStatus checks if the given status is a valid ProjectStatus
// Includes both legacy and new phase-based statuses for backward compatibility
func IsValidProjectStatus(status string) bool {
	switch ProjectStatus(status) {
	case ProjectStatusInitializing, ProjectStatusProcessing, ProjectStatusDone, ProjectStatusCompleted, ProjectStatusFailed, ProjectStatusPaused:
		return true
	default:
		return false
	}
}

// IsValidPhaseBasedProjectStatus checks if the given status is valid for phase-based progress messages
func IsValidPhaseBasedProjectStatus(status string) bool {
	switch PhaseBasedProjectStatus(status) {
	case PhaseBasedStatusInitializing, PhaseBasedStatusProcessing, PhaseBasedStatusDone, PhaseBasedStatusFailed:
		return true
	default:
		return false
	}
}

// IsValidJobStatus checks if the given status is a valid JobStatus
func IsValidJobStatus(status string) bool {
	switch JobStatus(status) {
	case JobStatusProcessing, JobStatusCompleted, JobStatusFailed, JobStatusPaused:
		return true
	default:
		return false
	}
}

// IsValidPlatform checks if the given platform is a valid Platform
func IsValidPlatform(platform string) bool {
	switch Platform(platform) {
	case PlatformTikTok, PlatformYouTube, PlatformInstagram:
		return true
	default:
		return false
	}
}
