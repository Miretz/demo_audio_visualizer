package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/inancgumus/screen"
	"github.com/mjibson/go-dsp/fft"
)

type FrequencyBand struct {
	min int
	max int
}

const numSamples = 4608
const bitSize = 64
const maxColumnWidth = 40
const visualChar = "█"
const magnitudeDivision = 12.0
const peakFalloff = 1.2

var mutex sync.Mutex

// most common frequency bands
var freqBands = []FrequencyBand{
	{0, 60}, {60, 90}, {90, 250}, {250, 300}, {300, 500}, // bass, low mid
	{500, 750}, {750, 1000}, {1000, 2000}, {2000, 3000}, {3000, 4000}, // mid, high mid
	{4000, 6000}, {6000, 8000}, {8000, 10000}, {10000, 20000}, {20000, 40000}} // high

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
	audioWave := make([]float64, numSamples)

	magnitudes := make([]float64, numSamples)
	freqSpectrum := make([]float64, len(freqBands))

	screen.Clear()

	for {
		_, err := d.Read(buf)
		if err != nil {
			break
		}
		p.Write(buf) // Playback

		// collect samples to the buffer - converting from byte to float64
		for i := 0; i < numSamples; i++ {
			audioWave[i], _ = strconv.ParseFloat(string(buf[i]), bitSize)
		}

		// get the fft for each sample
		fftOutput := fft.FFTReal(audioWave)

		// get the magnitudes
		for i := 0; i < numSamples; i++ {
			f := fftOutput[i]
			magnitudes[i] = math.Sqrt((real(f) * real(f)) + (imag(f) * imag(f)))
		}

		// get frequency per each sample and assign magnitude
		for i := 0; i < numSamples; i++ {
			frequency := i * d.SampleRate() / (numSamples / 2)
			for bandIndex := 0; bandIndex < len(freqBands); bandIndex++ {
				if frequency > freqBands[bandIndex].min && frequency <= freqBands[bandIndex].max {
					val := math.Max(magnitudes[i]/magnitudeDivision, 0.0)
					val = math.Min(maxColumnWidth, val)
					if val > freqSpectrum[bandIndex] {
						freqSpectrum[bandIndex] = val
					} else {
						freqSpectrum[bandIndex] = math.Max(freqSpectrum[bandIndex]-peakFalloff, 0)
					}
				}
			}
		}

		go updateScreen(freqSpectrum, &mutex)

	}

	return nil
}

func updateScreen(spectrum []float64, m *sync.Mutex) {

	m.Lock()

	screen.MoveTopLeft()

	// draw the columns to the console - will replace with a proper GUI library later
	for s := 0; s < len(freqBands); s++ {
		fmt.Print(strings.Repeat(visualChar, int(spectrum[s])))
		fmt.Print(strings.Repeat(" ", maxColumnWidth-int(spectrum[s])))
		fmt.Println()
	}

	m.Unlock()
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
