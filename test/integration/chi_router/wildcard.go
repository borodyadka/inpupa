package chirouter

// UserFilesInput — the params of the /users/{userID}/files/* route for the real-router test,
// including the bare trailing * wildcard, which chi records under the key "*".
type UserFilesInput struct {
	// @input:path userID
	UserID string
	// @input:path *
	Tail string
}
