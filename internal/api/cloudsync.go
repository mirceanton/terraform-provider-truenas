package api

// CloudSyncCredentialResponse represents a cloud sync credential from the API.
type CloudSyncCredentialResponse struct {
	ID         int64             `json:"id"`
	Name       string            `json:"name"`
	Provider   string            `json:"provider"`
	Attributes map[string]string `json:"attributes"`
}

// CloudSyncTaskResponse represents a cloud sync task from the API.
type CloudSyncTaskResponse struct {
	ID                 int64             `json:"id"`
	Description        string            `json:"description"`
	Path               string            `json:"path"`
	Credentials        int64             `json:"credentials"`
	Attributes         map[string]string `json:"attributes"`
	Schedule           ScheduleResponse  `json:"schedule"`
	Direction          string            `json:"direction"`
	TransferMode       string            `json:"transfer_mode"`
	Encryption         bool              `json:"encryption"`
	EncryptionPassword string            `json:"encryption_password,omitempty"`
	EncryptionSalt     string            `json:"encryption_salt,omitempty"`
	Snapshot           bool              `json:"snapshot"`
	Transfers          int64             `json:"transfers"`
	BWLimit            string            `json:"bwlimit,omitempty"`
	Exclude            []string          `json:"exclude"`
	FollowSymlinks     bool              `json:"follow_symlinks"`
	CreateEmptySrcDirs bool              `json:"create_empty_src_dirs"`
	Enabled            bool              `json:"enabled"`
	Job                *JobStatus        `json:"job,omitempty"`
}

// ScheduleResponse represents a cron schedule from the API.
type ScheduleResponse struct {
	Minute string `json:"minute"`
	Hour   string `json:"hour"`
	Dom    string `json:"dom"`
	Month  string `json:"month"`
	Dow    string `json:"dow"`
}

// JobStatus represents the last job status for a task.
type JobStatus struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
}
