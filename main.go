package main

import (
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

const numSamples = 4608
const defaultWindowWidth = 800
const defaultWindowHeight = 450
const spectrumSize = 80
const peakFalloff = 8.0
const bitSize = 64

var f *os.File
var d *mp3.Decoder
var c *oto.Context
var p *oto.Player

func play() error {

	buf := make([]byte, numSamples)
	audioWave := make([]float64, numSamples)
	freqSpectrum := make([]float64, spectrumSize)

	rl.SetConfigFlags(rl.FlagWindowResizable)
	var windowWidth int32 = defaultWindowWidth
	var windowHeight int32 = defaultWindowHeight

	rl.InitWindow(windowWidth, windowHeight, "Demo Audio Visualizer")
	rl.SetTargetFPS(60)

	isPlaying := false
	var nowPlayingText string

	for !rl.WindowShouldClose() {

		// Update on resize
		windowWidth = int32(rl.GetScreenWidth())
		windowHeight = int32(rl.GetScreenHeight())
		columnWidth := int32(windowWidth / spectrumSize)

		// handle file drag and drop
		if rl.IsFileDropped() {
			isOk, fileName, err := handleFileDrop()
			if err != nil {
				return err
			}
			if isOk {
				isPlaying = true
				nowPlayingText = "Now Playing: " + fileName
			}
		}

		// drawing code
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		if !isPlaying {
			drawDropzone(windowWidth, windowHeight)
		} else {

			// read buffer, update spectrum and play audio
			_, err := d.Read(buf)
			if err != nil {
				if err == io.EOF {
					isPlaying = false
				} else {
					return err
				}
			}
			updateSpectrumValues(buf, audioWave, d.SampleRate(), float64(windowHeight), freqSpectrum)
			p.Write(buf)

			for i, s := range freqSpectrum {
				rl.DrawRectangleGradientV(int32(i)*columnWidth, windowHeight-int32(s), columnWidth, int32(s), rl.Orange, rl.Green)
				rl.DrawRectangleLines(int32(i)*columnWidth, windowHeight-int32(s), columnWidth, int32(s), rl.Black)
			}
			rl.DrawText(nowPlayingText, 40, 40, 20, rl.White)
		}

		rl.EndDrawing()
	}

	defer rl.CloseWindow()
	defer closeFileHandlers()
	return nil
}

func handleFileDrop() (bool, string, error) {
	var count int32 = 0
	files := rl.GetDroppedFiles(&count)
	newFile := files[len(files)-1]
	rl.ClearDroppedFiles()
	if strings.HasSuffix(newFile, ".mp3") {
		fileName, err := updateFileHandlers(newFile)
		if err != nil {
			return false, err.Error(), err
		}
		return true, fileName, nil
	}
	return false, "Bad file type", nil
}

func drawDropzone(windowWidth, windowHeight int32) {
	var fontSize float32 = 16.0
	font := rl.GetFontDefault()
	message := "Drop your files to this window!"
	textPos := rl.Vector2{
		X: float32(windowWidth)/2.0 - rl.MeasureTextEx(font, message, fontSize, 2).X/2.0,
		Y: float32(windowHeight)/2.0 - fontSize/2.0}
	rl.DrawTextEx(font, message, textPos, fontSize, 2, rl.White)
	rl.DrawRectangleLines(20, 20, windowWidth-40, windowHeight-40, rl.LightGray)
}

func closeFileHandlers() {
	if p != nil {
		p.Close()
	}
	if c != nil {
		c.Close()
	}
	if f != nil {
		f.Close()
	}
}

func updateFileHandlers(filePath string) (string, error) {
	closeFileHandlers()
	var err error
	f, err = os.Open(filePath)
	if err != nil {
		return filePath, err
	}
	d, err = mp3.NewDecoder(f)
	if err != nil {
		return filePath, err
	}
	c, err = oto.NewContext(d.SampleRate(), 2, 2, 8192)
	if err != nil {
		return filePath, err
	}
	p = c.NewPlayer()

	fs, err := f.Stat()
	if err != nil {
		return filePath, err
	}
	return fs.Name(), nil
}

func updateSpectrumValues(buffer []byte, audioWave []float64, sampleRate int, maxValue float64, freqSpectrum []float64) {
	// collect samples to the buffer - converting from byte to float64
	for i := 0; i < numSamples; i++ {
		audioWave[i], _ = strconv.ParseFloat(string(buffer[i]), bitSize)
	}

	// apply window function
	window.Apply(audioWave, window.Blackman)

	// get the fft for each sample
	fftOutput := fft.FFTReal(audioWave)

	// get the magnitudes
	for i := 0; i < spectrumSize; i++ {
		fr := real(fftOutput[i])
		fi := imag(fftOutput[i])
		magnitude := math.Sqrt(fr*fr + fi*fi)
		val := math.Min(maxValue, math.Abs(magnitude))
		if freqSpectrum[i] > val {
			freqSpectrum[i] = math.Max(freqSpectrum[i]-peakFalloff, 0.0)
		} else {
			freqSpectrum[i] = (val + freqSpectrum[i]) / 2.0
		}
	}
}

func main() {
	if err := play(); err != nil {
		log.Fatal(err)
	}
}
