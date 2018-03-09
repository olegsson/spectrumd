// PulseAudio audio source from github.com/kierdavis/anna
package pulsesource

import (
	"pulse-simple"
)

type Source struct {
	stream *pulse.Stream
}

func New(sampleRate float64) (src Source, err error) {
	ss := pulse.SampleSpec{
		Format: pulse.SAMPLE_U8,
		Rate: uint32(sampleRate),
		Channels: 2,
	}

	s, err := pulse.Capture("", "stereo input", &ss)
	if err != nil {
		return Source{}, err
	}

	return Source{s}, nil
}

func (src Source) Read(left []float64, right []float64) (err error) {
	bytes := make([]byte, len(left)*2)
	_, err = src.stream.Read(bytes)
	if err != nil {
		return err
	}
	j := 0
	for i := range left {
		l, r := bytes[j], bytes[j+1]
		left[i] = float64(l) / 256.0 - .5
		right[i] = float64(r) / 256.0 - .5
		j += 2
	}

	return nil
}

func (src Source) Close() {
	src.stream.Free()
}
