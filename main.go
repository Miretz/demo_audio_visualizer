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
const windowWidth = 800
const windowHeight = 450
const spectrumSize = 40
const maxColumnHeight = 450
const columnWidth = 20
const peakFalloff = 8.0
const bitSize = 64

func play() error {

	buf := make([]byte, numSamples)
	freqSpectrum := make([]float64, spectrumSize)

	var f *os.File
	var d *mp3.Decoder
	var c *oto.Context
	var p *oto.Player
	var filePath string
	var fileName string

	isPlaying := false

	rl.InitWindow(windowWidth, windowHeight, "Demo Audio Visualizer")
	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {

		// handle file drag and drop
		if rl.IsFileDropped() {
			var count int32 = 0
			files := rl.GetDroppedFiles(&count)
			newFile := files[len(files)-1]
			rl.ClearDroppedFiles()

			if newFile != filePath && strings.HasSuffix(newFile, ".mp3") {
				filePath = newFile

				// clear the buffers
				for i := range buf {
					buf[i] = byte(0)
				}
				for i := range freqSpectrum {
					freqSpectrum[i] = 0.0
				}

				// close any open files
				if p != nil {
					p.Close()
				}
				if c != nil {
					c.Close()
				}
				if f != nil {
					f.Close()
				}

				// open the new file
				var err error
				f, err = os.Open(filePath)
				if err != nil {
					return err
				}
				fs, _ := f.Stat()
				fileName = fs.Name()
				defer f.Close()
				d, err = mp3.NewDecoder(f)
				if err != nil {
					return err
				}
				c, err = oto.NewContext(d.SampleRate(), 2, 2, 8192)
				if err != nil {
					return err
				}
				defer c.Close()
				p = c.NewPlayer()
				defer p.Close()
				isPlaying = true
			}
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		if !isPlaying {
			rl.DrawText("Drop your files to this window!", 220, 200, 20, rl.LightGray)
		} else {

			_, err := d.Read(buf)
			if err != nil {
				if err == io.EOF {
					isPlaying = false
				} else {
					return err
				}
			}
			updateSpectrumValues(buf, d.SampleRate(), freqSpectrum)
			p.Write(buf) // Playback

			rl.DrawCircleGradient(windowWidth/2, windowHeight/2, float32(freqSpectrum[0])/2.0, rl.DarkGray, rl.Black)

			for i, s := range freqSpectrum {
				rl.DrawRectangleGradientV(int32(i)*columnWidth, windowHeight-int32(s), columnWidth, int32(s), rl.Orange, rl.Green)
				rl.DrawRectangleLines(int32(i)*columnWidth, windowHeight-int32(s), columnWidth, int32(s), rl.Black)
			}

			rl.DrawText("Now Playing: "+fileName, 40, 40, 20, rl.White)
		}

		rl.EndDrawing()
	}

	rl.CloseWindow()

	return nil

}

func updateSpectrumValues(buffer []byte, sampleRate int, freqSpectrum []float64) {
	// collect samples to the buffer - converting from byte to float64
	audioWave := make([]float64, numSamples)
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
		val := math.Min(maxColumnHeight, math.Abs(magnitude))
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
