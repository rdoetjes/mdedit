package ui

import "testing"

func TestInsertVideo(t *testing.T) {
	h := newHarness(t, "")
	h.setText("Intro", 5, 5)
	h.run(h.a.cmds.video)
	d, ok := h.a.dialog.(*VideoDialog)
	if !ok {
		t.Fatal("Insert Video did not open its dialog")
	}
	d.id.SetText("not a video")
	d.submit()
	if d.err == "" || h.a.dialog == nil {
		t.Error("an invalid ID was accepted")
	}
	d.id.SetText("https://youtu.be/dQw4w9WgXcQ?si=x")
	d.title.SetText("My talk")
	d.submit()
	h.frame()
	h.frame()
	h.expect("Intro\n\n[![My talk](https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg)](https://www.youtube.com/watch?v=dQw4w9WgXcQ)\n")
	if h.a.dialog != nil {
		t.Error("dialog still open")
	}
	if b := h.a.doc.Blocks[1]; !b.Video {
		t.Errorf("inserted link is not rendered as a video: %+v", b)
	}
}
