package youtube

import "testing"

func TestParseID(t *testing.T) {
	const id = "dQw4w9WgXcQ"
	for _, in := range []string{
		id, " " + id + " ",
		"https://www.youtube.com/watch?v=" + id,
		"https://www.youtube.com/watch?v=" + id + "&t=42s",
		"https://youtu.be/" + id,
		"https://youtu.be/" + id + "?si=abc",
		"https://m.youtube.com/watch?v=" + id,
		"https://www.youtube.com/embed/" + id,
		"https://www.youtube.com/shorts/" + id,
		"https://img.youtube.com/vi/" + id + "/maxresdefault.jpg",
	} {
		if got, ok := ParseID(in); !ok || got != id {
			t.Errorf("ParseID(%q) = %q, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "short", "https://example.com/watch?v=" + id, "https://www.youtube.com/", "not a url at all"} {
		if got, ok := ParseID(in); ok {
			t.Errorf("ParseID(%q) = %q, want failure", in, got)
		}
	}
}

func TestURLs(t *testing.T) {
	const id = "dQw4w9WgXcQ"
	want := "[![Title](https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg)](https://www.youtube.com/watch?v=dQw4w9WgXcQ)"
	if got := Markdown(id, "Title"); got != want {
		t.Errorf("Markdown: %s", got)
	}
	if !IsThumbnail(ThumbnailURL(id)) || !IsThumbnail("https://i.ytimg.com/vi/"+id+"/hqdefault.jpg") {
		t.Error("thumbnail not recognised")
	}
	if IsThumbnail("https://example.com/vi/x.jpg") || IsThumbnail("http://img.youtube.com/vi/"+id+"/0.jpg") {
		t.Error("non-YouTube or plain-http image accepted")
	}
	if ThumbnailID(ThumbnailURL(id)) != id || !IsVideoLink(WatchURL(id)) || IsVideoLink("https://example.com") {
		t.Error("ThumbnailID / IsVideoLink")
	}
}
