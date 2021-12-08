package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/mjibson/go-dsp/fft"
)

type FrequencyBand struct {
	min int
	max int
}

func run() error {

	f, err := os.Open("song.mp3")
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

	const numSamples = 4608
	buf := make([]byte, numSamples)
	audioWave := make([]float64, numSamples)

	var freqBands = []FrequencyBand{
		{0, 60}, {60, 250}, {250, 500}, {500, 2000}, {2000, 4000}, {4000, 6000}, {20000, 40000}}

	const bitSize = 64
	fftCounter := 0
	const fftDelay = 1
	const maxColumnWidth = 30

	// TODO: cleanup
	for {
		_, err := d.Read(buf)
		if err != nil {
			break
		}
		p.Write(buf) // Playback

		fftCounter += 1
		if fftCounter == fftDelay {
			fftCounter = 0

			// collect samples to the buffer
			for i := 0; i < numSamples; i++ {
				audioWave[i], _ = strconv.ParseFloat(string(buf[i]), bitSize)
			}

			// get the fft for each sample
			fftOutput := fft.FFTReal(audioWave)
			freqSpectrum := make([]int, len(freqBands))

			// get the magnitudes
			var maxMagnitude float64 = -99999
			magnitudes := make([]float64, numSamples)
			for i := 0; i < numSamples; i++ {
				f := fftOutput[i]
				magnitudes[i] = math.Sqrt((real(f) * real(f)) + (imag(f) * imag(f)))
				if magnitudes[i] > maxMagnitude {
					maxMagnitude = magnitudes[i]
				}
			}

			// get peak frequency and assign value
			for i := 0; i < numSamples; i++ {
				frequency := i * d.SampleRate() / numSamples
				for bandIndex := 0; bandIndex < len(freqBands); bandIndex++ {
					if frequency > freqBands[bandIndex].min && frequency <= freqBands[bandIndex].max {
						freqSpectrum[bandIndex] = int(magnitudes[i] / maxMagnitude * maxColumnWidth)
					}
				}
			}

			fmt.Print("\033[H\033[2J")

			// draw the columns to the console - will replace with a proper GUI library later
			for s := 0; s < len(freqBands); s++ {
				for i := 0; i < freqSpectrum[s]; i++ {
					fmt.Print("█")
				}
				fmt.Println()
			}
		}

	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
