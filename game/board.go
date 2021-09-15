package game

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"code.rocketnine.space/tslocum/fibs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/llgcode/draw2d/draw2dimg"
)

type board struct {
	x, y int
	w, h int

	innerW, innerH int

	op *ebiten.DrawImageOptions

	backgroundImage *ebiten.Image

	Sprites *Sprites

	spaces     [][]*Sprite // Space contents
	spaceRects [][4]int

	dragging *Sprite
	moving   *Sprite // Moving automatically

	dragTouchId ebiten.TouchID
	touchIDs    []ebiten.TouchID

	spaceWidth           int
	barWidth             int
	triangleOffset       float64
	horizontalBorderSize int
	verticalBorderSize   int
	overlapSize          int

	lastDirection int

	s []string
	v []int

	drawFrame chan bool

	debug int // Print and draw debug information

	Client *fibs.Client

	dragX, dragY int
}

func NewBoard() *board {
	b := &board{
		barWidth:             100,
		triangleOffset:       float64(50),
		horizontalBorderSize: 50,
		verticalBorderSize:   25,
		overlapSize:          97,
		Sprites: &Sprites{
			sprites: make([]*Sprite, 30),
			num:     30,
		},
		spaces:     make([][]*Sprite, 26),
		spaceRects: make([][4]int, 26),
		s:          make([]string, 2),
		v:          make([]int, 42),
		drawFrame:  make(chan bool, 10),
	}

	for i := range b.Sprites.sprites {
		s := b.newSprite(i%2 == 1)

		b.Sprites.sprites[i] = s

		space := i
		if space > 25 {
			if space%2 == 0 {
				space = 0
			} else {
				space = 25
			}
		}

		b.spaces[space] = append(b.spaces[space], s)
	}

	go b.handleDraw()

	b.op = &ebiten.DrawImageOptions{}

	b.dragTouchId = -1

	return b
}

func (b *board) handleDraw() {
	drawFreq := time.Second / 144 // TODO
	lastDraw := time.Now()
	for v := range b.drawFrame {
		if !v {
			return
		}
		since := time.Since(lastDraw)
		if since < drawFreq {
			t := time.NewTimer(drawFreq - since)
		DELAYDRAW:
			for {
				select {
				case <-b.drawFrame:
					continue DELAYDRAW
				case <-t.C:
					break DELAYDRAW
				}
			}
		}
		ebiten.ScheduleFrame()
		lastDraw = time.Now()
	}
}

func (b *board) newSprite(white bool) *Sprite {
	s := &Sprite{}
	s.colorWhite = white
	s.w, s.h = imgCheckerWhite.Size()
	return s
}

func (b *board) updateBackgroundImage() {
	tableColor := color.RGBA{0, 102, 51, 255}
	frameColor := color.RGBA{65, 40, 14, 255}
	borderColor := color.RGBA{0, 0, 0, 255}
	faceColor := color.RGBA{120, 63, 25, 255}
	triangleA := color.RGBA{225.0, 188, 125, 255}
	triangleB := color.RGBA{120.0, 17.0, 0, 255}

	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}
	frameW := b.w - ((b.horizontalBorderSize - borderSize) * 2)
	innerW := b.w - (b.horizontalBorderSize * 2) // Outer board width (including frame)

	// Table
	box := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	img := ebiten.NewImageFromImage(box)
	img.Fill(tableColor)
	b.backgroundImage = ebiten.NewImageFromImage(img)

	// Frame
	box = image.NewRGBA(image.Rect(0, 0, frameW, b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(frameColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), 0)
	b.backgroundImage.DrawImage(img, b.op)

	// Face
	box = image.NewRGBA(image.Rect(0, 0, innerW, b.h-(b.verticalBorderSize*2)))
	img = ebiten.NewImageFromImage(box)
	img.Fill(faceColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

	// Bar
	box = image.NewRGBA(image.Rect(0, 0, b.barWidth, b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(frameColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64((b.w/2)-(b.barWidth/2)), 0)
	b.backgroundImage.DrawImage(img, b.op)

	// Draw triangles
	baseImg := image.NewRGBA(image.Rect(0, 0, b.w-(b.horizontalBorderSize*2), b.h-(b.verticalBorderSize*2)))
	gc := draw2dimg.NewGraphicContext(baseImg)
	for i := 0; i < 2; i++ {
		triangleTip := float64((b.h - (b.verticalBorderSize * 2)) / 2)
		if i == 0 {
			triangleTip -= b.triangleOffset
		} else {
			triangleTip += b.triangleOffset
		}
		for j := 0; j < 12; j++ {
			colorA := j%2 == 0
			if i == 1 {
				colorA = !colorA
			}

			if colorA {
				gc.SetFillColor(triangleA)
			} else {
				gc.SetFillColor(triangleB)
			}

			tx := b.spaceWidth * j
			ty := b.h * i
			if j >= 6 {
				tx += b.barWidth
			}
			gc.MoveTo(float64(tx), float64(ty))
			gc.LineTo(float64(tx+b.spaceWidth/2), triangleTip)
			gc.LineTo(float64(tx+b.spaceWidth), float64(ty))
			gc.Close()
			gc.Fill()
		}
	}
	img = ebiten.NewImageFromImage(baseImg)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

	// Border
	borderImage := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	gc = draw2dimg.NewGraphicContext(borderImage)
	gc.SetStrokeColor(borderColor)
	// - Outside left
	gc.SetLineWidth(2)
	gc.MoveTo(float64(1), float64(0))
	gc.LineTo(float64(1), float64(b.h))
	// - Center
	gc.SetLineWidth(2)
	gc.MoveTo(float64(frameW/2), float64(0))
	gc.LineTo(float64(frameW/2), float64(b.h))
	// - Outside right
	gc.MoveTo(float64(frameW), float64(0))
	gc.LineTo(float64(frameW), float64(b.h))
	gc.Close()
	gc.Stroke()
	// - Inside left
	gc.SetLineWidth(1)
	edge := float64((((innerW) - b.barWidth) / 2) + borderSize)
	gc.MoveTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.h-b.verticalBorderSize))
	gc.LineTo(float64(borderSize), float64(b.h-b.verticalBorderSize))
	gc.LineTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	// - Inside right
	edgeStart := float64((innerW / 2) + (b.barWidth / 2) + borderSize)
	edgeEnd := float64(innerW + borderSize)
	gc.MoveTo(float64(edgeStart), float64(b.verticalBorderSize))
	gc.LineTo(edgeEnd, float64(b.verticalBorderSize))
	gc.LineTo(edgeEnd, float64(b.h-b.verticalBorderSize))
	gc.LineTo(float64(edgeStart), float64(b.h-b.verticalBorderSize))
	gc.LineTo(float64(edgeStart), float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	img = ebiten.NewImageFromImage(borderImage)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), 0)
	b.backgroundImage.DrawImage(img, b.op)
}

func (b *board) ScheduleFrame() {
	b.drawFrame <- true
}

func (b *board) resetButtonRect() (int, int, int, int) {
	w := 200
	h := 75
	return (b.w - w) / 2, (b.h - h) / 2, w, h
}

func (b *board) draw(screen *ebiten.Image) {
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.x), float64(b.y))
	screen.DrawImage(b.backgroundImage, b.op)

	if b.debug == 2 {
		b.iterateSpaces(func(space int) {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			if space%2 == 0 {
				spaceImage.Fill(color.RGBA{50, 50, 50, 150})
			} else {
				spaceImage.Fill(color.RGBA{255, 255, 255, 150})
			}
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		})
	}

	drawSprite := func(sprite *Sprite) {
		x, y := float64(sprite.x), float64(sprite.y)
		if !sprite.toStart.IsZero() {
			progress := float64(time.Since(sprite.toStart)) / float64(sprite.toTime)
			if x == float64(sprite.toX) && y == float64(sprite.toY) {
				sprite.toStart = time.Time{}
				sprite.x = sprite.toX
				sprite.y = sprite.toY
			} else {
				if x < float64(sprite.toX) {
					x += (float64(sprite.toX) - x) * progress
					if x > float64(sprite.toX) {
						x = float64(sprite.toX)
					}
				} else if x > float64(sprite.toX) {
					x -= (x - float64(sprite.toX)) * progress
					if x < float64(sprite.toX) {
						x = float64(sprite.toX)
					}
				}

				if y < float64(sprite.toY) {
					y += (float64(sprite.toY) - y) * progress
					if y > float64(sprite.toY) {
						y = float64(sprite.toY)
					}
				} else if y > float64(sprite.toY) {
					y -= (y - float64(sprite.toY)) * progress
					if y < float64(sprite.toY) {
						y = float64(sprite.toY)
					}
				}
			}
			// Schedule another frame
			ebiten.ScheduleFrame()
		}
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(x, y)

		if sprite.colorWhite {
			screen.DrawImage(imgCheckerWhite, b.op)
		} else {
			screen.DrawImage(imgCheckerBlack, b.op)
		}
	}

	b.iterateSpaces(func(space int) {
		var numPieces int
		for i, sprite := range b.spaces[space] {
			if sprite == b.dragging || sprite == b.moving {
				continue
			}
			numPieces++

			drawSprite(sprite)

			var overlayText string
			if i > 5 {
				overlayText = fmt.Sprintf("%d", numPieces)
			}
			if sprite.premove {
				if overlayText != "" {
					overlayText += " "
				}
				overlayText += "%"
			}
			if overlayText == "" {
				continue
			}

			labelColor := color.RGBA{255, 255, 255, 255}
			if sprite.colorWhite {
				labelColor = color.RGBA{0, 0, 0, 255}
			}

			bounds := text.BoundString(mplusNormalFont, overlayText)
			overlayImage := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
			text.Draw(overlayImage, overlayText, mplusNormalFont, 0, bounds.Dy(), labelColor)

			x, y, w, h := b.stackSpaceRect(space, numPieces-1)
			x += (w / 2) - (bounds.Dx() / 2)
			y += (h / 2) - (bounds.Dy() / 2)
			x, y = b.offsetPosition(x, y)

			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(overlayImage, b.op)
		}
	})

	// Draw hover overlay

	if b.dragging != nil {
		dx, dy := b.dragX, b.dragY

		x, y := ebiten.CursorPosition()
		if x != 0 || y != 0 {
			dx, dy = x, y
		}

		space := b.spaceAt(dx, dy)
		if space >= 0 {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			spaceImage.Fill(color.RGBA{255, 255, 255, 50})
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		}
	}

	// Draw opponent name and dice

	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}

	playerColor := color.White
	opponentColor := color.Black
	if b.v[fibs.StatePlayerColor] == -1 {
		playerColor = color.Black
		opponentColor = color.White
	}

	drawLabel := func(label string, labelColor color.Color, border bool, borderColor color.Color) *ebiten.Image {
		bounds := text.BoundString(mplusNormalFont, label)

		w := int(float64(bounds.Dx()) * 1.5)
		h := int(float64(bounds.Dy()) * 2)

		baseImg := image.NewRGBA(image.Rect(0, 0, w, h))
		// Draw border
		if border {
			gc := draw2dimg.NewGraphicContext(baseImg)
			gc.SetLineWidth(5)
			gc.SetStrokeColor(borderColor)
			gc.MoveTo(float64(0), float64(0))
			gc.LineTo(float64(w), 0)
			gc.LineTo(float64(w), float64(h))
			gc.LineTo(float64(0), float64(h))
			gc.Close()
			gc.Stroke()
		}

		img := ebiten.NewImageFromImage(baseImg)
		text.Draw(img, label, mplusNormalFont, (w-bounds.Dx())/2, int(float64(h-(bounds.Max.Y/2))*0.75), labelColor)

		return img
	}

	if b.s[fibs.StateOpponentName] != "" {
		label := fmt.Sprintf("%s  %d %d", b.s[fibs.StateOpponentName], b.v[fibs.StateOpponentDice1], b.v[fibs.StateOpponentDice2])

		img := drawLabel(label, opponentColor, b.v[fibs.StateTurn] != b.v[fibs.StatePlayerColor], opponentColor)
		bounds := img.Bounds()

		x := ((b.innerW - borderSize) / 4) - (bounds.Dx() / 2)
		y := (b.innerH / 2) - (bounds.Dy() / 2)
		x, y = b.offsetPosition(x, y)
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)
	}

	// Draw player name and dice

	if b.s[fibs.StatePlayerName] != "" {
		label := fmt.Sprintf("%s  %d %d", b.s[fibs.StatePlayerName], b.v[fibs.StatePlayerDice1], b.v[fibs.StatePlayerDice2])

		img := drawLabel(label, playerColor, b.v[fibs.StateTurn] == b.v[fibs.StatePlayerColor], playerColor)
		bounds := img.Bounds()

		x := ((b.innerW / 4) * 3) - (bounds.Dx() / 2)
		y := (b.innerH / 2) - (bounds.Dy() / 2)
		x, y = b.offsetPosition(x, y)
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)

	}

	if len(b.Client.Board.GetPreMoves()) > 0 {
		x, y, w, h := b.resetButtonRect()
		baseImg := image.NewRGBA(image.Rect(0, 0, w, h))

		gc := draw2dimg.NewGraphicContext(baseImg)
		gc.SetLineWidth(5)
		gc.SetStrokeColor(color.Black)
		gc.MoveTo(0, 0)
		gc.LineTo(float64(w), 0)
		gc.LineTo(float64(w), float64(h))
		gc.LineTo(0, float64(h))
		gc.Close()
		gc.Stroke()
		img := ebiten.NewImage(w, h)
		img.Fill(color.RGBA{225.0, 188, 125, 255})
		img.DrawImage(ebiten.NewImageFromImage(baseImg), nil)

		label := "Reset"
		bounds := text.BoundString(mplusNormalFont, label)
		text.Draw(img, label, mplusNormalFont, (w-bounds.Dx())/2, (h+(bounds.Dy()/2))/2, color.Black)

		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)
	}

	// Draw moving sprite
	if b.moving != nil {
		drawSprite(b.moving)
	}

	// Draw dragged sprite
	if b.dragging != nil {
		drawSprite(b.dragging)
	}

	if b.debug == 2 {
		b.iterateSpaces(func(space int) {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			br := ""
			if b.bottomRow(space) {
				br = "B"
			}
			ebitenutil.DebugPrint(spaceImage, fmt.Sprintf(" %d %s", space, br))
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Scale(2, 2)
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		})
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}
	const stackAllowance = 0.97 // TODO configurable

	b.x, b.y, b.w, b.h = x, y, w, h

	b.horizontalBorderSize = 0

	b.triangleOffset = float64(b.h-(b.verticalBorderSize*2)) / 15

	for {
		b.verticalBorderSize = 7 // TODO configurable

		b.spaceWidth = (b.w - (b.horizontalBorderSize * 2)) / 13

		b.barWidth = b.spaceWidth

		b.overlapSize = (((b.h - (b.verticalBorderSize * 2)) - (int(b.triangleOffset) * 2)) / 2) / 5
		o := int(float64(b.spaceWidth) * stackAllowance)
		if b.overlapSize >= o {
			b.overlapSize = o
			break
		}

		b.horizontalBorderSize++
	}

	extraSpace := b.w - (b.spaceWidth * 12)
	largeBarWidth := int(float64(b.spaceWidth) * 1.25)
	if extraSpace >= largeBarWidth {
		b.barWidth = largeBarWidth
	}

	b.horizontalBorderSize = ((b.w - (b.spaceWidth * 12)) - b.barWidth) / 2
	if b.horizontalBorderSize < 0 {
		b.horizontalBorderSize = 0
	}

	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}
	b.innerW = b.w - (b.horizontalBorderSize * 2)
	b.innerH = b.h - (b.verticalBorderSize * 2)

	loadAssets(b.spaceWidth)

	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerWhite.Size()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.positionCheckers()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	return b.x + x + b.horizontalBorderSize, b.y + y + b.verticalBorderSize
}

func (b *board) positionCheckers() {
	for space := 0; space < 26; space++ {
		sprites := b.spaces[space]

		for i := range sprites {
			s := sprites[i]
			if b.dragging == s {
				continue
			}

			x, y, w, _ := b.stackSpaceRect(space, i)
			s.x, s.y = b.offsetPosition(x, y)
			// Center piece in space
			s.x += (w - s.w) / 2
		}
	}

	b.ScheduleFrame()
}

func (b *board) spriteAt(x, y int) *Sprite {
	space := b.spaceAt(x, y)
	if space == -1 {
		return nil
	}
	pieces := b.spaces[space]
	if len(pieces) == 0 {
		return nil
	}
	return pieces[len(pieces)-1]
}

func (b *board) spaceAt(x, y int) int {
	for i := 0; i < 26; i++ {
		sx, sy, sw, sh := b.spaceRect(i)
		sx, sy = b.offsetPosition(sx, sy)
		if x >= sx && x <= sx+sw && y >= sy && y <= sy+sh {
			return i
		}
	}
	return -1
}

func (b *board) iterateSpaces(f func(space int)) {
	for space := 0; space <= 25; space++ {
		f(space)
	}
}

func (b *board) translateSpace(space int) int {
	if b.v[fibs.StateDirection] == -1 {
		// Spaces range from 24 - 1.
		if space == 0 || space == 25 {
			space = 25 - space
		} else if space <= 12 {
			space = 12 + space
		} else {
			space = space - 12
		}
	}
	return space
}

func (b *board) setSpaceRects() {
	var x, y, w, h int
	for i := 0; i < 26; i++ {
		trueSpace := i

		space := b.translateSpace(i)
		if !b.bottomRow(trueSpace) {
			y = 0
		} else {
			y = (b.h / 2) - b.verticalBorderSize
		}

		w = b.spaceWidth

		var hspace int // horizontal space
		var add int
		if space == 0 {
			hspace = 6
			w = b.barWidth
		} else if space == 25 {
			hspace = 6
			w = b.barWidth
		} else if space <= 6 {
			hspace = space - 1
		} else if space <= 12 {
			hspace = space - 1
			add = b.barWidth
		} else if space <= 18 {
			hspace = 24 - space
			add = b.barWidth
		} else {
			hspace = 24 - space
		}

		x = (b.spaceWidth * hspace) + add

		h = (b.h - (b.verticalBorderSize * 2)) / 2

		b.spaceRects[trueSpace] = [4]int{x, y, w, h}
	}
}

// relX, relY
func (b *board) spaceRect(space int) (x, y, w, h int) {
	rect := b.spaceRects[space]
	return rect[0], rect[1], rect[2], rect[3]
}

func (b *board) bottomRow(space int) bool {
	bottomStart := 1
	bottomEnd := 12
	bottomBar := 25
	if b.v[fibs.StateDirection] == 1 {
		bottomStart = 13
		bottomEnd = 24
		bottomBar = 0
	}
	return space == bottomBar || (space >= bottomStart && space <= bottomEnd)
}

// relX, relY
func (b *board) stackSpaceRect(space int, stack int) (x, y, w, h int) {
	x, y, w, h = b.spaceRect(space)

	// Stack pieces
	osize := float64(stack)
	var o int
	if stack > 4 {
		osize = 3.5
	}
	if b.bottomRow(space) {
		osize += 1.0
	}
	o = int(osize * float64(b.overlapSize))
	if !b.bottomRow(space) {
		y += o
	} else {
		y = y + (h - o)
	}

	w, h = b.spaceWidth, b.spaceWidth
	if space == 0 || space == 25 {
		w = b.barWidth
	}

	return x, y, w, h
}

func (b *board) SetState(s []string, v []int) {
	copy(b.s, s)
	copy(b.v, v)
	b.ProcessState()
}

func (b *board) ProcessState() {
	v := b.v

	if b.lastDirection != v[fibs.StateDirection] {
		b.setSpaceRects()
	}
	b.lastDirection = v[fibs.StateDirection]

	b.Sprites = &Sprites{}
	b.spaces = make([][]*Sprite, 26)
	for space := 0; space < 26; space++ {
		spaceValue := v[fibs.StateBoardSpace0+space]

		white := spaceValue > 0
		if spaceValue == 0 {
			white = v[fibs.StatePlayerColor] == 1
		}

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}

		var preMovesTo = b.Client.Board.Premoveto[space]
		var preMovesFrom = b.Client.Board.Premovefrom[space]

		for i := 0; i < abs+(preMovesTo-preMovesFrom); i++ {
			s := b.newSprite(white)
			if i >= abs {
				s.colorWhite = v[fibs.StatePlayerColor] == 1
				s.premove = true
			}
			b.spaces[space] = append(b.spaces[space], s)
			b.Sprites.sprites = append(b.Sprites.sprites, s)
		}
	}
	b.Sprites.num = len(b.Sprites.sprites)

	b.positionCheckers()
}

func (b *board) _movePiece(sprite *Sprite, from int, to int, speed int, pause bool) {
	moveTime := time.Second / time.Duration(speed)
	pauseTime := 750 * time.Millisecond

	b.moving = sprite

	space := to // Immediately go to target space

	stack := len(b.spaces[space])
	if stack == 1 && sprite.colorWhite != b.spaces[space][0].colorWhite {
		stack = 0 // Hit
	} else if space != to {
		stack++
	}

	x, y, _, _ := b.stackSpaceRect(space, stack)
	x, y = b.offsetPosition(x, y)

	sprite.toX = x
	sprite.toY = y
	sprite.toTime = moveTime
	sprite.toStart = time.Now()
	ebiten.ScheduleFrame()
	time.Sleep(sprite.toTime)
	sprite.x = x
	sprite.y = y

	if pause {
		log.Println("PAUSE ", pauseTime)
		time.Sleep(pauseTime)
		log.Println("UNPAUSE")
	}

	homeSpace := b.Client.Board.PlayerHomeSpace()
	if b.v[fibs.StateTurn] != b.v[fibs.StatePlayerColor] {
		homeSpace = 25 - homeSpace
	}

	if to != homeSpace {
		b.spaces[to] = append(b.spaces[to], sprite)
	}
	for i, s := range b.spaces[from] {
		if s == sprite {
			b.spaces[from] = append(b.spaces[from][:i], b.spaces[from][i+1:]...)
			break
		}
	}
	b.moving = nil

	b.ScheduleFrame()
}

// Do not call directly
func (b *board) movePiece(from int, to int) {
	pieces := b.spaces[from]
	if len(pieces) == 0 {
		log.Printf("%d-%d: NO PIECES AT SPACE %d", from, to, from)
		return
	}

	sprite := pieces[len(pieces)-1]

	var moveAfter *Sprite
	if len(b.spaces[to]) == 1 {
		if sprite.colorWhite != b.spaces[to][0].colorWhite {
			moveAfter = b.spaces[to][0]
		}
	}

	b._movePiece(sprite, from, to, 1, moveAfter == nil)
	if moveAfter != nil {
		toBar := 0
		if b.v[fibs.StateDirection] == -1 {
			toBar = 25
		}
		if b.v[fibs.StateTurn] != b.v[fibs.StatePlayerColor] {
			toBar = 25 - toBar // TODO how is this determined?
		}
		b._movePiece(moveAfter, to, toBar, 1, true)
	}
	log.Println("FINISH MOVE PIECE", from, to)
}

func (b *board) update() {
	if b.Client == nil {
		return
	}

	if b.dragging == nil {
		// TODO allow grabbing multiple pieces by grabbing further down the stack

		handleReset := func(x, y int) bool {
			if len(b.Client.Board.GetPreMoves()) > 0 {
				rx, ry, rw, rh := b.resetButtonRect()
				if x >= rx && x <= rx+rw && y >= ry && y <= ry+rh {
					b.Client.Board.ResetPreMoves()
					b.ProcessState()
					return true
				}
			}
			return false
		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if b.dragging == nil {
				x, y := ebiten.CursorPosition()

				handled := handleReset(x, y)
				if !handled {
					s := b.spriteAt(x, y)
					if s != nil {
						b.dragging = s
					}
				}
			}
		}

		b.touchIDs = inpututil.AppendJustPressedTouchIDs(b.touchIDs[:0])
		for _, id := range b.touchIDs {
			x, y := ebiten.TouchPosition(id)

			handled := handleReset(x, y)
			if !handled {
				b.dragX, b.dragY = x, y

				s := b.spriteAt(x, y)
				if s != nil {
					b.dragging = s
					b.dragTouchId = id
				}
			}
		}
	}

	x, y := ebiten.CursorPosition()
	if b.dragTouchId != -1 {
		x, y = ebiten.TouchPosition(b.dragTouchId)

		if x != 0 || y != 0 { // 0,0 is returned when the touch is released
			b.dragX, b.dragY = x, y
		} else {
			x, y = b.dragX, b.dragY
		}
	}

	var dropped *Sprite
	if b.dragTouchId == -1 {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			dropped = b.dragging
			b.dragging = nil
		}
	} else if inpututil.IsTouchJustReleased(b.dragTouchId) {
		dropped = b.dragging
		b.dragging = nil
	}
	if dropped != nil {
		// TODO allow dragging anywhere outside of board to bear off
		// allow dragging on to bar to bear off

		index := b.spaceAt(x, y)
		if index >= 0 {
			if b.Client != nil {
				for space, pieces := range b.spaces {
					for _, piece := range pieces {
						if piece == dropped {
							if space != index {
								b.Client.Board.SetSelection(1, space)
								b.Client.Board.AddPreMove(space, index)
								b.ProcessState()
							}
							break
						}
					}
				}
			}
		}
		b.positionCheckers()
	}

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}
