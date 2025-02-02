package graphics

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
	"go.uber.org/zap"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// ThemeManager is an interface to fetch the current theme (colors, etc.).
type ThemeManager interface {
	GetTheme() Theme
}

// ChartGenerator creates charts using Gonum/Plot and optionally styles them from a theme.
type ChartGenerator struct {
	mutex        sync.RWMutex
	logger       *zap.Logger
	themeManager ThemeManager // Fetches colors from the current theme
}

// NewChartGenerator initializes a ChartGenerator.
func NewChartGenerator(logger *zap.Logger, tm ThemeManager) (*ChartGenerator, error) {
	return &ChartGenerator{
		logger:       logger,
		themeManager: tm,
	}, nil
}

// GenerateLineChart creates a line chart with the given title and data.
func (cg *ChartGenerator) GenerateLineChart(title string, xData []string, yData []float64) (image.Image, error) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	cg.logger.Info("Generating Line Chart", zap.String("title", title))

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	// Apply theming if available
	if cg.themeManager != nil {
		theme := cg.themeManager.GetTheme()
		applyThemeToPlot(p, theme, cg.logger)
	}

	xValues := cg.parseXData(xData)
	pts := make(plotter.XYs, len(yData))
	for i := range yData {
		pts[i].X = xValues[i]
		pts[i].Y = yData[i]
	}

	line, err := plotter.NewLine(pts)
	if err != nil {
		cg.logger.Error("Error creating line plotter", zap.Error(err))
		return nil, fmt.Errorf("failed to create line plotter: %w", err)
	}

	if cg.themeManager != nil {
		theme := cg.themeManager.GetTheme()
		line.Color = colorToGonum(theme.TextColor)
	}

	p.Add(line)
	p.Legend.Add("Data", line)

	return cg.renderPlot(p)
}

// GenerateBarChart creates a bar chart with the given title and data.
func (cg *ChartGenerator) GenerateBarChart(title string, xData []string, yData []float64) (image.Image, error) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	cg.logger.Info("Generating Bar Chart", zap.String("title", title))

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Categories"
	p.Y.Label.Text = "Values"

	if cg.themeManager != nil {
		theme := cg.themeManager.GetTheme()
		applyThemeToPlot(p, theme, cg.logger)
	}

	w := vg.Points(20)
	bars, err := plotter.NewBarChart(plotter.Values(yData), w)
	if err != nil {
		cg.logger.Error("Error creating bar chart", zap.Error(err))
		return nil, fmt.Errorf("failed to create bar chart: %w", err)
	}

	if cg.themeManager != nil {
		theme := cg.themeManager.GetTheme()
		bars.Color = colorToGonum(theme.SelectionColor)
	}

	p.Add(bars)
	p.NominalX(xData...)

	return cg.renderPlot(p)
}

// GeneratePieChart creates a pie chart with the given title and data.
func (cg *ChartGenerator) GeneratePieChart(title string, labels []string, values []float64) (image.Image, error) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	cg.logger.Info("Generating Pie Chart", zap.String("title", title))

	p := plot.New()
	p.Title.Text = title

	wedges := make([]*plotter.Wedge, len(values))
	for i, v := range values {
		wedges[i] = &plotter.Wedge{
			Theta:  2 * vg.Rad(v),
			Radius: 100,
			FillColor: colorToGonum(color.RGBA{
				R: uint8(50 * (i + 1)),
				G: uint8(50 * (len(values) - i)),
				B: 200, A: 255}),
		}
	}

	for _, wedge := range wedges {
		p.Add(wedge)
	}

	return cg.renderPlot(p)
}

// renderPlot renders the Plot to an in-memory image.
func (cg *ChartGenerator) renderPlot(p *plot.Plot) (image.Image, error) {
	canvas := vgimg.New(vg.Points(600), vg.Points(400))
	dc := draw.New(canvas)
	if err := p.Draw(dc); err != nil {
		cg.logger.Error("Error drawing plot", zap.Error(err))
		return nil, fmt.Errorf("failed to draw plot: %w", err)
	}
	return canvas.Image(), nil
}

// DisplayChart uses Raylib to display the chart in a window.
func (cg *ChartGenerator) DisplayChart(img image.Image) {
	rl.InitWindow(800, 600, "Chart Display")
	defer rl.CloseWindow()

	bgColor := rl.RayWhite
	if cg.themeManager != nil {
		theme := cg.themeManager.GetTheme()
		bgColor = colorToRaylib(theme.BackgroundColor)
	}

	texture := cg.imageToTexture(img)
	defer rl.UnloadTexture(texture)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(bgColor)
		rl.DrawTexture(texture, 0, 0, rl.White)
		rl.EndDrawing()
	}
}

// imageToTexture converts an image.Image to a Raylib Texture2D.
func (cg *ChartGenerator) imageToTexture(img image.Image) rl.Texture2D {
	bounds := img.Bounds()
	width, height := int32(bounds.Dx()), int32(bounds.Dy())
	data := make([]uint8, width*height*4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			idx := (y*width + x) * 4
			data[idx] = uint8(r >> 8)
			data[idx+1] = uint8(g >> 8)
			data[idx+2] = uint8(b >> 8)
			data[idx+3] = uint8(a >> 8)
		}
	}

	image := rl.GenImageColor(width, height, rl.Blank)
	image.Data = data
	return rl.LoadTextureFromImage(image)
}

// Helper to apply theme colors to a Gonum plot.
func applyThemeToPlot(p *plot.Plot, theme Theme, logger *zap.Logger) {
	bg := colorToGonum(theme.BackgroundColor)
	textColor := colorToGonum(theme.TextColor)
	p.BackgroundColor = bg
	p.Title.TextStyle.Color = textColor
	p.X.Color = textColor
	p.Y.Color = textColor
	logger.Debug("Applied theme to plot")
}

// colorToRaylib converts color.RGBA to Raylib's rl.Color.
func colorToRaylib(c color.RGBA) rl.Color {
	return rl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// colorToGonum converts color.RGBA to Gonum's color.Color.
func colorToGonum(c color.RGBA) color.Color {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}
