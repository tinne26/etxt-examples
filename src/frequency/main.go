package main

import "os"
import "fmt"
import "math"
import "image"
import "image/color"
import "unicode"

import "github.com/hajimehoshi/ebiten/v2"
import "golang.org/x/image/font/sfnt"

import "github.com/tinne26/badcolor"
import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/fonts/liberation/lbrtserif"

// for chinese texts, you would have to pass a font (e.g. NotoSansSC-Regular)
// manually as the first argument to the program. if you are unfamiliar with
// the language to analyze, wikipedia is a good source of text in any language
const SampleText = "To be, or not to be, that is the question: whether 'tis nobler in the mind to suffer the slings and arrows of outrageous fortune, or to take arms against a sea of troubles and by opposing end them?\n To die: to sleep; no more; and, by a sleep to say we end the heart-ache and the thousand natural shocks that flesh is heir to, 'tis a consummation devoutly to be wish'd. To die, to sleep;\n To sleep: perchance to dream: ay, there's the rub; for in that sleep of death what dreams may come when we have shuffled off this mortal coil, must give us pause."
const Normalized = false

// color map
const NumColors = 32
var ColdColor = color.RGBA{0, 192, 192, 255}
var MidColor = color.RGBA{192, 192, 128, 255}
var HotColor  = color.RGBA{255, 96, 0, 255}
var BackgroundColor = color.RGBA{23, 18, 25, 255}
var ColorMap  = make([]color.RGBA, NumColors)

// ---- helper function to compute glyph frequencies ----

func ComputeSampleTextFreqs(text *etxt.Renderer) map[sfnt.GlyphIndex]float32 {
	// get rune counts
	glyphCounts := make(map[sfnt.GlyphIndex]int, 64)
	for _, codePoint := range SampleText {
		// ignore control codes
		if codePoint < ' ' { continue }

		// translate rune to glyph index
		glyphIndex := text.Glyph().GetRuneIndex(unicode.ToLower(codePoint))
		glyphCounts[glyphIndex] += 1
	}

	// add uppercases with same values as lowercases
	for _, codePoint := range SampleText {
		// ignore control codes
		if codePoint < ' ' { continue }
		if codePoint == ' ' {
			glyphCounts[text.Glyph().GetRuneIndex(' ')] = 1
		}

		// copy lowecase data into uppercase
		lower := unicode.ToLower(codePoint)
		if lower == codePoint { continue }
		upperIndex := text.Glyph().GetRuneIndex(codePoint)
		lowerIndex := text.Glyph().GetRuneIndex(lower)
		glyphCounts[upperIndex] = glyphCounts[lowerIndex]
	}
	
	// get normalization count
	var normCount int
	if Normalized {
		for _, count := range glyphCounts {
			normCount += count
		}
		normCount = int(float64(normCount)*(1.0/3.0))
	} else { // using max count instead
		for _, count := range glyphCounts {
			normCount = max(normCount, count)
		}
	}
	if normCount == 0 { panic("no glyphs in text") }

	// obtain frequencies relative to maxCount
	freqs := make(map[sfnt.GlyphIndex]float32)
	for glyphIndex, count := range glyphCounts {
		freqs[glyphIndex] = min(float32(count)/float32(normCount), 1.0)
	}

	return freqs
}

// ---- Ebitengine's Game interface implementation ----

type Game struct {
	text *etxt.Renderer
	freqs map[sfnt.GlyphIndex]float32
}

func (self *Game) Layout(winWidth int, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	return nil
}

func (self *Game) Draw(canvas *ebiten.Image) {
	// background color
	canvas.Fill(BackgroundColor)
	
	// get screen center position and text content
	bounds := canvas.Bounds() // assumes origin (0, 0)
	width, height := bounds.Dx(), bounds.Dy()
	x, y := width/2, height/2
	margin := int(ebiten.DeviceScaleFactor()*48.0)

	// draw colors on the bottom
	colorWidth := int(ebiten.DeviceScaleFactor()*8)
	colorHeight := int(ebiten.DeviceScaleFactor()*18)
	colorsWidth := colorWidth*NumColors
	cx := x - colorsWidth/2
	cy := height - (colorHeight + colorHeight/2)
	for i := 0; i < NumColors; i++ {
		rect := image.Rect(cx, cy, cx + colorWidth, cy + colorHeight)
		canvas.SubImage(rect).(*ebiten.Image).Fill(ColorMap[i])
		cx += colorWidth
	}

	// draw the text
	self.text.DrawWithWrap(canvas, SampleText, x, y, width - margin)
}

func (self *Game) GlyphDrawFunc(target etxt.Target, glyphIndex sfnt.GlyphIndex, origin fract.Point) {
	// get glyph frequency and convert to color
	freq, found := self.freqs[glyphIndex]
	if !found { panic("unexpected glyph index") }
	if freq <= 0 || freq > 1.0 { panic("unexpected frequency value") }
	colorIndex := max(int(freq*NumColors) - 1, 0)

	// set color, get mask and draw
	self.text.SetColor(ColorMap[colorIndex])
	mask := self.text.Glyph().LoadMask(glyphIndex, origin)
	self.text.Glyph().DrawMask(target, mask, origin)
}

// ---- main function ----

func main() {
	// accept font argument
	sfntFont := lbrtserif.Font()
	if len(os.Args) == 2 {
		var fontName string
		var err error
		sfntFont, fontName, err = font.ParseFromPath(os.Args[1])
		if err != nil { panic(err) }
		fmt.Printf("Custom font loaded: %s\n", fontName)
	}
	
	// create text renderer, set the font and cache
	renderer := etxt.NewRenderer()
	renderer.SetFont(sfntFont)
	renderer.Utils().SetCache8MiB()
	
	// adjust main text style properties
	renderer.SetAlign(etxt.Center)
	renderer.SetSize(20)

	// initialize color map
	badcolor.OklabGradient(ColorMap[0 : NumColors/2], ColdColor, MidColor)
	badcolor.OklabGradient(ColorMap[NumColors/2 : ], MidColor, HotColor)

	// instance game struct and set custom draw function
	game := &Game{
		text: renderer,
		freqs: ComputeSampleTextFreqs(renderer),
	}
	renderer.Glyph().SetDrawFunc(game.GlyphDrawFunc)

	// screenshot key
	err := os.Setenv("EBITENGINE_SCREENSHOT_KEY", "q")
	if err != nil { panic(err) }

	// set up Ebitengine and start the game
	ebiten.SetWindowTitle("etxt-examples/frequency")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	err = ebiten.RunGame(game)
	if err != nil { panic(err) }
}
