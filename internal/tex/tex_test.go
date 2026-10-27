package tex

import "testing"

var corpus = []string{
	`x^2 + y^2 = z^2`,
	`e^{i\pi} + 1 = 0`,
	`\frac{a}{b} + \dfrac{1}{1+\frac{1}{x}}`,
	`\sqrt{2}, \sqrt[3]{x^3+y^3}, \sqrt{\frac{a}{b}}`,
	`\sum_{i=1}^{n} i = \frac{n(n+1)}{2}`,
	`\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}`,
	`\lim_{x \to 0} \frac{\sin x}{x} = 1`,
	`\left( \frac{a}{b} \right)^2 \left[ x \right] \left\{ \frac{1}{2} \right\}`,
	`\left\langle \psi \middle| \phi \right\rangle`,
	`\begin{pmatrix} a & b \\ c & d \end{pmatrix} \begin{bmatrix} 1 & 0 & 0 \\ 0 & 1 & 0 \\ 0 & 0 & 1 \end{bmatrix}`,
	`f(x) = \begin{cases} x^2 & x \ge 0 \\ -x & x < 0 \end{cases}`,
	`\begin{aligned} a &= b + c \\ &= d \end{aligned}`,
	`\hat{x} \bar{y} \vec{v} \tilde{n} \dot{a} \ddot{b} \overline{AB} \underline{x} \widehat{abc}`,
	`\mathbb{R}^n \mathcal{L} \mathbf{v} \mathfrak{g} \mathrm{d}x \boldsymbol{\alpha}`,
	`\alpha\beta\gamma\Gamma\Delta\Omega\varepsilon\varphi`,
	`a \le b \ne c \approx d \in E \subseteq F \Rightarrow G \iff H`,
	`\binom{n}{k} = \frac{n!}{k!(n-k)!}`,
	`\text{if } x > 0 \text{ then } y`,
	`\overbrace{a+b}^{n} \underbrace{c+d}_{m}`,
	`\Big( \big[ x \big] \Big)`,
	`\nabla \times \mathbf{E} = -\frac{\partial \mathbf{B}}{\partial t}`,
	`x' + f''(x)`,
	`a \\ b \\ c`,
	`a &= 1 \\ bb &= 2`,
	// Malformed input must not panic.
	`\frac{a}`, `x^`, `\left( x`, `\right)`, `}{`, `\begin{matrix} a & b`, `\unknowncommand{x}`,
	`\sqrt[`, `^^__`, `\`, `{{{`, `\end{x}`, `&&\\\\`, `\left. \right.`,
}

func TestCorpus(t *testing.T) {
	for _, src := range corpus {
		for _, display := range []bool{true, false} {
			b := Layout(src, display)
			if b == nil {
				t.Fatalf("%q: nil box", src)
			}
			if b.W < 0 || b.H < 0 || b.D < 0 || b.W > 100 || b.H > 20 {
				t.Errorf("%q: odd size %+v", src, *b)
			}
		}
	}
}

func TestErrorsShown(t *testing.T) {
	b := Layout(`\nosuch x`, true)
	red := 0
	for _, it := range b.Items {
		if it.Error {
			red++
		}
	}
	if red != len(`\nosuch`) {
		t.Errorf("want %d red glyphs, got %d", len(`\nosuch`), red)
	}
}

func TestSpacing(t *testing.T) {
	// a+b has medium spaces around +; -b (unary) has none.
	ab := Layout("a+b", false).W
	a, plus, b := Layout("a", false).W, Layout("{+}", false).W, Layout("b", false).W
	if d := ab - (a + plus + b); d < 0.4 || d > 0.5 {
		t.Errorf("binary spacing %.3f, want 2×4mu = 0.444", d)
	}
	if d := Layout("-b", false).W - (Layout("{-}", false).W + b); d > 0.01 {
		t.Errorf("unary minus spacing %.3f, want 0", d)
	}
}

func TestPipeSpacing(t *testing.T) {
	// |x| = 1 needs thick spaces around '=': '|' is ordinary.
	for i := 0; i < 20; i++ { // the class used to depend on map order
		runeClasses = nil
		w := Layout("|x|=1", false).W
		parts := Layout("|x|", false).W + Layout("{=}", false).W + Layout("1", false).W
		if d := w - parts; d < 0.5 || d > 0.6 {
			t.Fatalf("spacing around = after |x|: %.3f, want 2×5mu = 0.556", d)
		}
		cacheMu.Lock()
		cache = map[cacheKey]*Box{}
		cacheMu.Unlock()
	}
}
