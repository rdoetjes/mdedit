// Package youtube knows the YouTube URLs used by GitHub-style video links:
//
//	[![Title](https://img.youtube.com/vi/ID/maxresdefault.jpg)](https://www.youtube.com/watch?v=ID)
package youtube

import (
	"net/url"
	"regexp"
	"strings"
)

// idRe matches a video ID: 11 characters of URL-safe base64.
var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ValidID reports whether s looks like a video ID.
func ValidID(s string) bool { return idRe.MatchString(s) }

// ThumbnailURL is the full-resolution still of a video.
func ThumbnailURL(id string) string { return "https://img.youtube.com/vi/" + id + "/maxresdefault.jpg" }

// FallbackThumbnailURL is a still that exists for every video; the
// full-resolution one is missing for some older or low-resolution videos.
func FallbackThumbnailURL(id string) string {
	return "https://img.youtube.com/vi/" + id + "/hqdefault.jpg"
}

// WatchURL is the page that plays a video.
func WatchURL(id string) string { return "https://www.youtube.com/watch?v=" + id }

// Markdown returns the GitHub-style link showing the video's still.
func Markdown(id, title string) string {
	return "[![" + title + "](" + ThumbnailURL(id) + ")](" + WatchURL(id) + ")"
}

// ParseID extracts a video ID from an ID or a YouTube link (watch, short,
// embed and youtu.be forms).
func ParseID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if ValidID(s) {
		return s, true
	}
	u, err := url.Parse(s)
	if err != nil || !isYouTubeHost(u.Hostname()) {
		return "", false
	}
	id := idFromURL(u)
	return id, ValidID(id)
}

func idFromURL(u *url.URL) string {
	if v := u.Query().Get("v"); v != "" {
		return v
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch {
	case u.Hostname() == "youtu.be" && len(parts) == 1:
		return parts[0]
	case len(parts) >= 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "vi" || parts[0] == "live"):
		return parts[1]
	}
	return ""
}

func isYouTubeHost(host string) bool {
	switch strings.TrimPrefix(strings.ToLower(host), "www.") {
	case "youtube.com", "m.youtube.com", "youtu.be", "youtube-nocookie.com", "img.youtube.com", "i.ytimg.com":
		return true
	}
	return false
}

// IsVideoLink reports whether a link target plays a YouTube video.
func IsVideoLink(s string) bool {
	u, err := url.Parse(s)
	return err == nil && isYouTubeHost(u.Hostname()) && ValidID(idFromURL(u))
}

// IsThumbnail reports whether src is a YouTube still image, which mdedit
// may download. Other remote images are not fetched.
func IsThumbnail(src string) bool {
	u, err := url.Parse(src)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return (host == "img.youtube.com" || host == "i.ytimg.com") && strings.HasPrefix(u.Path, "/vi/")
}

// ThumbnailID returns the video ID of a thumbnail URL.
func ThumbnailID(src string) string {
	u, err := url.Parse(src)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "vi" {
		return parts[1]
	}
	return ""
}
