package main

import (
	"fmt"
	"io"
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
	"github.com/mjibson/go-dsp/window"
)

const numSamples = 64
const maxColumnWidth = 40
const visualChar = "█"
const emptyChar = " "
const peakFalloff = 0.9
const bitSize = 64
const spectrumSize = 15

var mutex sync.Mutex

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

	screen.Clear()

	for {
		_, err := d.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		updateSpectrumValues(buf, d.SampleRate(), freqSpectrum)
		go updateScreen(filename, freqSpectrum, &mutex)
		p.Write(buf) // Playback
	}

	screen.Clear()

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
		val := math.Min(maxColumnWidth, math.Abs(magnitude))
		if freqSpectrum[i] > val {
			freqSpectrum[i] = math.Max(freqSpectrum[i]-peakFalloff, 0.0)
		} else {
			freqSpectrum[i] = (val + freqSpectrum[i]) / 2.0
		}
	}
}

func updateScreen(filename string, spectrum []float64, m *sync.Mutex) {

	m.Lock()

	screen.MoveTopLeft()

	// draw the columns to the console - will replace with a proper GUI library later
	fmt.Println("> Now playing: ", filename)
	for i, s := range spectrum {
		is := int(s)
		if is == 0 {
			fmt.Println()
		} else {
			fmt.Printf("%02d %02d %s%s\n", i, is,
				strings.Repeat(visualChar, is),
				strings.Repeat(emptyChar, maxColumnWidth-is))
		}
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
