package gateway

import "regexp"

// Claude Code notes each image of the message it is given in a line of its
// own after the message ("[Image: source: …/images/1.png]", where it keeps
// them, or the size it showed a large one at). A client that takes the
// images out of its history (#1382) leaves the notes alone in it, and a run
// started anew, told the conversation again, has the model read one as an
// image sent with the turn to answer, its file missing. The path is where
// Claude Code kept the image, which no model can read, so every note goes
// as shown earlier: the same told whatever the request carries, as the
// history's blocks have to be for their cache mark to hold.
var imageNoteLine = regexp.MustCompile(`(?m)^\[Image:[^\n]*$`)

// rewriteImageNotes is s with each of Claude Code's image notes a line
// saying only that the image was shown earlier.
func rewriteImageNotes(s string) string {
	return imageNoteLine.ReplaceAllString(s, "[Image: shown earlier]")
}
