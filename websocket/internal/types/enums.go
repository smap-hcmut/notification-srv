package types

// ProjectStatus represents the current status of a project
type ProjectStatus string

const (
	ProjectStatusProcessing ProjectStatus = "PROCESSING" // Project is being processed (crawling and analysis)
	ProjectStatusCompleted  ProjectStatus = "COMPLETED"  // Project finished successfully
	ProjectStatusFailed     ProjectStatus = "FAILED"     // Project encountered fatal error
	ProjectStatusPaused     ProjectStatus = "PAUSED"     // Project temporarily stopped
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
func IsValidProjectStatus(status string) bool {
	switch ProjectStatus(status) {
	case ProjectStatusProcessing, ProjectStatusCompleted, ProjectStatusFailed, ProjectStatusPaused:
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