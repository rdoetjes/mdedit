package ui

import (
	"image"
	"math"
	"sync/atomic"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"mdedit/internal/mdmodel"
	"mdedit/internal/render"
)

// Preview renders a parsed document.
type Preview struct {
	doc     *mdmodel.Document
	BaseDir string
	// OnLink is called when a link is clicked.
	OnLink func(url string)
	// Invalidate asks for a new frame; it may be called from any goroutine.
	Invalidate func()
	// imagesChanged is set when a downloaded image arrives.
	imagesChanged atomic.Bool

	list   widget.List
	cache  []*render.Layout
	cacheW int
	cacheE float32
	clicks []gesture.Click
	text   *textPainter
	images map[string]*imageEntry

	// Per-frame geometry.
	ctx      render.Ctx
	width    int
	pad      int
	contentW int
	ox       float32
}

func NewPreview(shaper *text.Shaper) *Preview {
	p := &Preview{
		text:   newTextPainter(shaper),
		images: map[string]*imageEntry{},
	}
	p.list.Axis = layout.Vertical
	p.ctx = render.Ctx{Theme: &render.Light, Measure: p.text.measure, ImageSize: p.imageSize, VideoControls: true}
	return p
}

// SetDocument replaces the displayed document, keeping the scroll position.
func (p *Preview) SetDocument(doc *mdmodel.Document) {
	p.doc = doc
	p.cache = make([]*render.Layout, len(doc.Blocks))
	for len(p.clicks) < len(doc.Blocks) {
		p.clicks = append(p.clicks, gesture.Click{})
	}
}

// SetTheme changes the colours of the preview.
func (p *Preview) SetTheme(t *render.Theme) {
	p.ctx.Theme = t
	if p.doc != nil {
		p.cache = make([]*render.Layout, len(p.doc.Blocks))
	}
}

// ScrollToBlock scrolls so block i is at the top.
func (p *Preview) ScrollToBlock(i int) {
	p.list.Position.First = i + 1 // index 0 is the top padding
	p.list.Position.Offset = 0
	p.list.Position.BeforeEnd = true
}

// EnsureVisible scrolls to block i if it is outside the visible range.
func (p *Preview) EnsureVisible(i int) {
	pos := p.list.Position
	first, last := pos.First-1, pos.First+pos.Count-2
	if pos.Count == 0 || i < first || i > last {
		p.ScrollToBlock(i)
	}
}

const maxContentWidth = unit.Dp(880)

func (p *Preview) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	fillRect(gtx, p.ctx.Theme.Bg, image.Rectangle{Max: gtx.Constraints.Max})
	if p.doc == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	p.measure(gtx)
	return material.List(th, &p.list).Layout(gtx, len(p.doc.Blocks)+2, p.listItem)
}

// measure works out the content column for this frame, dropping cached
// layouts if its width or the font size changed.
func (p *Preview) measure(gtx layout.Context) {
	p.width = gtx.Constraints.Max.X
	p.pad = gtx.Dp(32)
	if p.width < gtx.Dp(500) {
		p.pad = gtx.Dp(12)
	}
	// Leave room for the scrollbar.
	p.contentW = max(min(p.width-2*p.pad-gtx.Dp(10), gtx.Dp(maxContentWidth)), gtx.Dp(100))
	p.ox = float32((p.width - p.contentW) / 2)
	em := float32(gtx.Sp(16))
	if p.contentW != p.cacheW || em != p.cacheE || p.imagesChanged.Swap(false) {
		p.cache = make([]*render.Layout, len(p.doc.Blocks))
		p.cacheW, p.cacheE = p.contentW, em
	}
	p.ctx.Em = em
}

// listItem lays out list element i: padding at both ends, blocks between.
func (p *Preview) listItem(gtx layout.Context, i int) layout.Dimensions {
	if i == 0 || i == len(p.doc.Blocks)+1 {
		return layout.Dimensions{Size: image.Pt(p.width, p.pad)}
	}
	return p.block(gtx, i-1)
}

func (p *Preview) block(gtx layout.Context, i int) layout.Dimensions {
	l := p.blockLayout(i)
	size := image.Pt(p.width, int(math.Ceil(float64(l.Height))))
	p.handleClicks(gtx, i, l)
	p.paint(gtx, l, p.ox)
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	p.clicks[i].Add(gtx.Ops)
	area.Pop()
	return layout.Dimensions{Size: size}
}

// blockLayout returns the (cached) layout of block i.
func (p *Preview) blockLayout(i int) *render.Layout {
	if l := p.cache[i]; l != nil {
		return l
	}
	var prev *mdmodel.Block
	if i > 0 {
		prev = &p.doc.Blocks[i-1]
	}
	l := p.ctx.Block(&p.doc.Blocks[i], prev, float32(p.contentW))
	p.cache[i] = l
	return l
}

// handleClicks follows links clicked in block i.
func (p *Preview) handleClicks(gtx layout.Context, i int, l *render.Layout) {
	for {
		ev, ok := p.clicks[i].Update(gtx.Source)
		if !ok {
			return
		}
		if ev.Kind != gesture.KindClick {
			continue
		}
		pos := f32.Pt(float32(ev.Position.X)-p.ox, float32(ev.Position.Y))
		if url := linkAt(l, pos); url != "" && p.OnLink != nil {
			p.OnLink(url)
		}
	}
}

// linkAt returns the target of the link at pos, if any.
func linkAt(l *render.Layout, pos f32.Point) string {
	for _, pr := range l.Prims {
		if pr.Kind == render.PLink && pos.X >= pr.X && pos.X < pr.X+pr.W && pos.Y >= pr.Y && pos.Y < pr.Y+pr.H {
			return pr.URL
		}
	}
	return ""
}

func (p *Preview) imageSize(src string) (int, int, bool) {
	img := p.image(src)
	if img == nil {
		return 0, 0, false
	}
	return img.size.X, img.size.Y, true
}

func rnd(v float32) int { return int(math.Round(float64(v))) }
