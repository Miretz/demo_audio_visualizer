package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"

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

func play(filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		return err
	}

	c, err := oto.NewContext(d.SampleRate(), 2, 2, 8192)
	if err != nil {
		return err
	}
	defer c.Close()

	p := c.NewPlayer()
	defer p.Close()

	buf := make([]byte, numSamples)
	freqSpectrum := make([]float64, spectrumSize)

	rl.InitWindow(windowWidth, windowHeight, "Demo Audio Visualizere")
	rl.SetTargetFPS(60)
	for !rl.WindowShouldClose() {

		_, err := d.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		updateSpectrumValues(buf, d.SampleRate(), freqSpectrum)
		p.Write(buf) // Playback

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		for i, s := range freqSpectrum {
			rl.DrawRectangleGradientV(int32(i)*columnWidth, windowHeight-int32(s), columnWidth, int32(s), rl.Orange, rl.Green)
		}

		rl.DrawText("Now Playing: "+filename, 190, 200, 20, rl.LightGray)
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
	if len(os.Args) != 2 {
		fmt.Printf("Usage: demo_audio_visualizer filename.mp3\n")
		return
	}
	filename := os.Args[1]
	if err := play(filename); err != nil {
		log.Fatal(err)
	}

}
