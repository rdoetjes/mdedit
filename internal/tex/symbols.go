package tex

// class is a TeX atom class; it decides the spacing between atoms.
type class int8

const (
	clsOrd class = iota
	clsOp
	clsBin
	clsRel
	clsOpen
	clsClose
	clsPunct
	clsInner
	clsNone // spaces and style changes: no atom spacing
)

type symbol struct {
	r   rune
	cls class
}

// symbols maps control sequences to characters.
var symbols = map[string]symbol{
	// Lowercase Greek (italic in maths).
	"alpha": {'α', clsOrd}, "beta": {'β', clsOrd}, "gamma": {'γ', clsOrd}, "delta": {'δ', clsOrd},
	"epsilon": {'ϵ', clsOrd}, "varepsilon": {'ε', clsOrd}, "zeta": {'ζ', clsOrd}, "eta": {'η', clsOrd},
	"theta": {'θ', clsOrd}, "vartheta": {'ϑ', clsOrd}, "iota": {'ι', clsOrd}, "kappa": {'κ', clsOrd},
	"varkappa": {'ϰ', clsOrd}, "lambda": {'λ', clsOrd}, "mu": {'μ', clsOrd}, "nu": {'ν', clsOrd},
	"xi": {'ξ', clsOrd}, "omicron": {'ο', clsOrd}, "pi": {'π', clsOrd}, "varpi": {'ϖ', clsOrd},
	"rho": {'ρ', clsOrd}, "varrho": {'ϱ', clsOrd}, "sigma": {'σ', clsOrd}, "varsigma": {'ς', clsOrd},
	"tau": {'τ', clsOrd}, "upsilon": {'υ', clsOrd}, "phi": {'ϕ', clsOrd}, "varphi": {'φ', clsOrd},
	"chi": {'χ', clsOrd}, "psi": {'ψ', clsOrd}, "omega": {'ω', clsOrd},
	// Uppercase Greek (upright).
	"Gamma": {'Γ', clsOrd}, "Delta": {'Δ', clsOrd}, "Theta": {'Θ', clsOrd}, "Lambda": {'Λ', clsOrd},
	"Xi": {'Ξ', clsOrd}, "Pi": {'Π', clsOrd}, "Sigma": {'Σ', clsOrd}, "Upsilon": {'Υ', clsOrd},
	"Phi": {'Φ', clsOrd}, "Psi": {'Ψ', clsOrd}, "Omega": {'Ω', clsOrd},

	// Ordinary symbols.
	"infty": {'∞', clsOrd}, "partial": {'∂', clsOrd}, "nabla": {'∇', clsOrd}, "forall": {'∀', clsOrd},
	"exists": {'∃', clsOrd}, "nexists": {'∄', clsOrd}, "emptyset": {'∅', clsOrd}, "varnothing": {'∅', clsOrd},
	"hbar": {'ℏ', clsOrd}, "hslash": {'ℏ', clsOrd}, "ell": {'ℓ', clsOrd}, "Re": {'ℜ', clsOrd}, "Im": {'ℑ', clsOrd},
	"aleph": {'ℵ', clsOrd}, "beth": {'ℶ', clsOrd}, "wp": {'℘', clsOrd}, "angle": {'∠', clsOrd},
	"triangle": {'△', clsOrd}, "prime": {'′', clsOrd}, "degree": {'°', clsOrd}, "neg": {'¬', clsOrd},
	"lnot": {'¬', clsOrd}, "top": {'⊤', clsOrd}, "bot": {'⊥', clsOrd}, "flat": {'♭', clsOrd},
	"natural": {'♮', clsOrd}, "sharp": {'♯', clsOrd}, "clubsuit": {'♣', clsOrd}, "diamondsuit": {'♢', clsOrd},
	"heartsuit": {'♡', clsOrd}, "spadesuit": {'♠', clsOrd}, "imath": {'ı', clsOrd}, "jmath": {'ȷ', clsOrd},
	"backslash": {'\\', clsOrd}, "vert": {'|', clsOrd}, "Vert": {'‖', clsOrd}, "|": {'‖', clsOrd},
	"ldots": {'…', clsInner}, "dots": {'…', clsInner}, "cdots": {'⋯', clsInner}, "vdots": {'⋮', clsOrd},
	"ddots": {'⋱', clsInner}, "dotsc": {'…', clsInner}, "dotsb": {'⋯', clsInner}, "checkmark": {'✓', clsOrd},
	"square": {'□', clsOrd}, "blacksquare": {'■', clsOrd}, "qed": {'∎', clsOrd}, "S": {'§', clsOrd},
	"P": {'¶', clsOrd}, "dag": {'†', clsOrd}, "ddag": {'‡', clsOrd}, "copyright": {'©', clsOrd},
	"$": {'$', clsOrd}, "%": {'%', clsOrd}, "&": {'&', clsOrd}, "#": {'#', clsOrd}, "_": {'_', clsOrd},
	"{": {'{', clsOpen}, "}": {'}', clsClose}, "lbrace": {'{', clsOpen}, "rbrace": {'}', clsClose},

	// Binary operators.
	"pm": {'±', clsBin}, "mp": {'∓', clsBin}, "times": {'×', clsBin}, "div": {'÷', clsBin},
	"cdot": {'⋅', clsBin}, "ast": {'∗', clsBin}, "star": {'⋆', clsBin}, "circ": {'∘', clsBin},
	"bullet": {'∙', clsBin}, "cup": {'∪', clsBin}, "cap": {'∩', clsBin}, "setminus": {'∖', clsBin},
	"wedge": {'∧', clsBin}, "land": {'∧', clsBin}, "vee": {'∨', clsBin}, "lor": {'∨', clsBin},
	"oplus": {'⊕', clsBin}, "ominus": {'⊖', clsBin}, "otimes": {'⊗', clsBin}, "oslash": {'⊘', clsBin},
	"odot": {'⊙', clsBin}, "uplus": {'⊎', clsBin}, "sqcup": {'⊔', clsBin}, "sqcap": {'⊓', clsBin},
	"wr": {'≀', clsBin}, "diamond": {'⋄', clsBin}, "bigtriangleup": {'△', clsBin}, "bigtriangledown": {'▽', clsBin},
	"triangleleft": {'◃', clsBin}, "triangleright": {'▹', clsBin}, "dagger": {'†', clsBin},
	"ddagger": {'‡', clsBin}, "amalg": {'⨿', clsBin}, "cdotp": {'⋅', clsPunct},

	// Relations.
	"leq": {'≤', clsRel}, "le": {'≤', clsRel}, "geq": {'≥', clsRel}, "ge": {'≥', clsRel},
	"neq": {'≠', clsRel}, "ne": {'≠', clsRel}, "leqslant": {'⩽', clsRel}, "geqslant": {'⩾', clsRel},
	"ll": {'≪', clsRel}, "gg": {'≫', clsRel}, "approx": {'≈', clsRel}, "equiv": {'≡', clsRel},
	"sim": {'∼', clsRel}, "simeq": {'≃', clsRel}, "cong": {'≅', clsRel}, "propto": {'∝', clsRel},
	"in": {'∈', clsRel}, "notin": {'∉', clsRel}, "ni": {'∋', clsRel}, "subset": {'⊂', clsRel},
	"supset": {'⊃', clsRel}, "subseteq": {'⊆', clsRel}, "supseteq": {'⊇', clsRel}, "subsetneq": {'⊊', clsRel},
	"supsetneq": {'⊋', clsRel}, "sqsubseteq": {'⊑', clsRel}, "sqsupseteq": {'⊒', clsRel},
	"perp": {'⟂', clsRel}, "parallel": {'∥', clsRel}, "mid": {'∣', clsRel}, "nmid": {'∤', clsRel},
	"vdash": {'⊢', clsRel}, "dashv": {'⊣', clsRel}, "models": {'⊨', clsRel}, "asymp": {'≍', clsRel},
	"doteq": {'≐', clsRel}, "prec": {'≺', clsRel}, "succ": {'≻', clsRel}, "preceq": {'⪯', clsRel},
	"succeq": {'⪰', clsRel}, "smile": {'⌣', clsRel}, "frown": {'⌢', clsRel}, "bowtie": {'⋈', clsRel},
	"coloneqq": {'≔', clsRel}, "triangleq": {'≜', clsRel}, "lessgtr": {'≶', clsRel}, "lesssim": {'≲', clsRel},
	"gtrsim": {'≳', clsRel}, "nleq": {'≰', clsRel}, "ngeq": {'≱', clsRel}, "nsim": {'≁', clsRel},
	"ncong": {'≇', clsRel}, "nsubseteq": {'⊈', clsRel}, "nsupseteq": {'⊉', clsRel},
	// Arrows.
	"to": {'→', clsRel}, "rightarrow": {'→', clsRel}, "leftarrow": {'←', clsRel}, "gets": {'←', clsRel},
	"leftrightarrow": {'↔', clsRel}, "Rightarrow": {'⇒', clsRel}, "Leftarrow": {'⇐', clsRel},
	"Leftrightarrow": {'⇔', clsRel}, "implies": {'⟹', clsRel}, "impliedby": {'⟸', clsRel}, "iff": {'⟺', clsRel},
	"longrightarrow": {'⟶', clsRel}, "longleftarrow": {'⟵', clsRel}, "longleftrightarrow": {'⟷', clsRel},
	"Longrightarrow": {'⟹', clsRel}, "Longleftarrow": {'⟸', clsRel}, "Longleftrightarrow": {'⟺', clsRel},
	"mapsto": {'↦', clsRel}, "longmapsto": {'⟼', clsRel}, "uparrow": {'↑', clsRel}, "downarrow": {'↓', clsRel},
	"updownarrow": {'↕', clsRel}, "Uparrow": {'⇑', clsRel}, "Downarrow": {'⇓', clsRel},
	"nearrow": {'↗', clsRel}, "searrow": {'↘', clsRel}, "swarrow": {'↙', clsRel}, "nwarrow": {'↖', clsRel},
	"hookrightarrow": {'↪', clsRel}, "hookleftarrow": {'↩', clsRel}, "rightharpoonup": {'⇀', clsRel},
	"leftharpoonup": {'↼', clsRel}, "rightleftharpoons": {'⇌', clsRel}, "leadsto": {'⇝', clsRel},
	"twoheadrightarrow": {'↠', clsRel}, "rightsquigarrow": {'⇝', clsRel},

	// Delimiters.
	"langle": {'⟨', clsOpen}, "rangle": {'⟩', clsClose}, "lvert": {'|', clsOpen}, "rvert": {'|', clsClose},
	"lVert": {'‖', clsOpen}, "rVert": {'‖', clsClose}, "lfloor": {'⌊', clsOpen}, "rfloor": {'⌋', clsClose},
	"lceil": {'⌈', clsOpen}, "rceil": {'⌉', clsClose}, "lbrack": {'[', clsOpen}, "rbrack": {']', clsClose},
	"ulcorner": {'⌜', clsOpen}, "urcorner": {'⌝', clsClose}, "llcorner": {'⌞', clsOpen}, "lrcorner": {'⌟', clsClose},

	// Punctuation.
	"colon": {':', clsPunct}, "ldotp": {'.', clsPunct},
}

// bigOps are large operators; limits reports whether their scripts go above
// and below in display style.
var bigOps = map[string]struct {
	r      rune
	limits bool
}{
	"sum": {'∑', true}, "prod": {'∏', true}, "coprod": {'∐', true},
	"int": {'∫', false}, "iint": {'∬', false}, "iiint": {'∭', false}, "oint": {'∮', false},
	"oiint": {'∯', false}, "bigcup": {'⋃', true}, "bigcap": {'⋂', true}, "bigvee": {'⋁', true},
	"bigwedge": {'⋀', true}, "bigoplus": {'⨁', true}, "bigotimes": {'⨂', true}, "bigodot": {'⨀', true},
	"biguplus": {'⨄', true}, "bigsqcup": {'⨆', true},
}

// functions are operator names set upright; limits as for bigOps.
var functions = map[string]bool{
	"sin": false, "cos": false, "tan": false, "cot": false, "sec": false, "csc": false,
	"arcsin": false, "arccos": false, "arctan": false, "sinh": false, "cosh": false, "tanh": false,
	"coth": false, "log": false, "ln": false, "lg": false, "exp": false, "arg": false, "deg": false,
	"dim": false, "hom": false, "ker": false, "det": true, "gcd": true, "lim": true, "liminf": true,
	"limsup": true, "max": true, "min": true, "sup": true, "inf": true, "Pr": true, "argmax": true,
	"argmin": true, "tr": false, "rank": false, "sgn": false,
}

var functionNames = map[string]string{"liminf": "lim inf", "limsup": "lim sup", "argmax": "arg max", "argmin": "arg min"}

// Spaces in em.
var spaces = map[string]float32{
	",": 3.0 / 18, "thinspace": 3.0 / 18, ":": 4.0 / 18, ">": 4.0 / 18, "medspace": 4.0 / 18,
	";": 5.0 / 18, "thickspace": 5.0 / 18, "!": -3.0 / 18, "negthinspace": -3.0 / 18,
	"quad": 1, "qquad": 2, " ": 0.25, "enspace": 0.5,
}

// Font styles for letters.
type fontStyle uint8

const (
	fItalic fontStyle = iota // default for letters
	fRoman
	fBold
	fBoldItalic
	fBlackboard
	fScript
	fFraktur
	fSans
	fMono
)

var fontCommands = map[string]fontStyle{
	"mathrm": fRoman, "mathup": fRoman, "rm": fRoman, "mathit": fItalic, "mathbf": fBold, "bf": fBold,
	"boldsymbol": fBoldItalic, "bm": fBoldItalic, "mathbb": fBlackboard, "mathcal": fScript,
	"mathscr": fScript, "mathfrak": fFraktur, "mathsf": fSans, "mathtt": fMono,
}

// styled maps a character to its form in the given mathematical alphabet.
func styled(r rune, f fontStyle) rune {
	upper := r >= 'A' && r <= 'Z'
	lower := r >= 'a' && r <= 'z'
	digit := r >= '0' && r <= '9'
	greekLower := r >= 'α' && r <= 'ω'
	var exceptions map[rune]rune
	var up, lo, dig rune
	switch f {
	case fRoman:
		return r
	case fItalic:
		if r == 'h' {
			return 'ℎ'
		}
		if greekLower {
			return 0x1D6FC + (r - 'α')
		}
		if v, ok := italicGreekVariants[r]; ok {
			return v
		}
		up, lo = 0x1D434, 0x1D44E
	case fBold:
		up, lo, dig = 0x1D400, 0x1D41A, 0x1D7CE
	case fBoldItalic:
		up, lo = 0x1D468, 0x1D482
		if greekLower {
			return 0x1D736 + (r - 'α')
		}
	case fBlackboard:
		up, lo, dig = 0x1D538, 0x1D552, 0x1D7D8
		exceptions = map[rune]rune{'C': 'ℂ', 'H': 'ℍ', 'N': 'ℕ', 'P': 'ℙ', 'Q': 'ℚ', 'R': 'ℝ', 'Z': 'ℤ'}
	case fScript:
		up, lo = 0x1D49C, 0x1D4B6
		exceptions = map[rune]rune{'B': 'ℬ', 'E': 'ℰ', 'F': 'ℱ', 'H': 'ℋ', 'I': 'ℐ', 'L': 'ℒ', 'M': 'ℳ', 'R': 'ℛ',
			'e': 'ℯ', 'g': 'ℊ', 'o': 'ℴ'}
	case fFraktur:
		up, lo = 0x1D504, 0x1D51E
		exceptions = map[rune]rune{'C': 'ℭ', 'H': 'ℌ', 'I': 'ℑ', 'R': 'ℜ', 'Z': 'ℨ'}
	case fSans:
		up, lo, dig = 0x1D5A0, 0x1D5BA, 0x1D7E2
	case fMono:
		up, lo, dig = 0x1D670, 0x1D68A, 0x1D7F6
	}
	if v, ok := exceptions[r]; ok {
		return v
	}
	switch {
	case upper && up != 0:
		return up + (r - 'A')
	case lower && lo != 0:
		return lo + (r - 'a')
	case digit && dig != 0:
		return dig + (r - '0')
	}
	return r
}

var italicGreekVariants = map[rune]rune{
	'ϵ': 0x1D716, 'ϑ': 0x1D717, 'ϰ': 0x1D718, 'ϕ': 0x1D719, 'ϱ': 0x1D71A, 'ϖ': 0x1D71B,
}

// Accent characters.
var accents = map[string]rune{
	"hat": 'ˆ', "widehat": 'ˆ', "tilde": '˜', "widetilde": '˜', "dot": '˙', "ddot": '¨',
	"breve": '˘', "check": 'ˇ', "acute": '´', "grave": '`', "mathring": '˚',
}

// delimiterNames maps \left/\right arguments given as commands.
var delimiterNames = map[string]rune{
	"{": '{', "}": '}', "lbrace": '{', "rbrace": '}', "langle": '⟨', "rangle": '⟩',
	"lvert": '|', "rvert": '|', "vert": '|', "lVert": '‖', "rVert": '‖', "Vert": '‖', "|": '‖',
	"lfloor": '⌊', "rfloor": '⌋', "lceil": '⌈', "rceil": '⌉', "lbrack": '[', "rbrack": ']',
	"backslash": '\\', "uparrow": '↑', "downarrow": '↓', "updownarrow": '↕',
}
