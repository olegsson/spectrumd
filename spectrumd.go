package main

import (
    "github.com/mjibson/go-dsp/fft"
    "spectrumd/pulsesource"
    "math/cmplx"
    "math"
    "fmt"
    "net"
    "os"
    "encoding/json"
    "runtime"
    "sync"
)

const SampleRate = 44100
const InputBufferSize = 1024
const AmplitudeScale = 10000

func getBand(frequency float64) (band uint32) {
    switch {
    case frequency < 60:
        return 0
    case 60 <= frequency && frequency < 250:
        return 1
    case 250 <= frequency && frequency < 500:
        return 2
    case 500 <= frequency && frequency < 2000:
        return 3
    case 2000 <= frequency && frequency < 4000:
        return 4
    case 4000 <= frequency && frequency < 20000:
        return 5
    default:
        return 6
    }
}

type Spectrum struct {
    AmpsLeft []int32 `json:"ampsl"`
    AmpsRight []int32 `json:"ampsr"`
    MaxLeft float64 `json:"maxl"`
    MaxRight float64 `json:"maxr"`
    mux sync.Mutex
}

func MinMax(array []float64) (float64, float64) {
    max := array[0]
    min := array[0]
    for _, value := range array {
        if max < value {
            max = value
        }
        if min > value {
            min = value
        }
    }
    return min, max
}

func (s *Spectrum) Set(l []int32, r []int32, maxl float64, maxr float64) {
    s.mux.Lock()
    s.AmpsLeft = l
    s.AmpsRight = r
    s.MaxLeft = maxl
    s.MaxRight = maxr
    s.mux.Unlock()
}

func (s *Spectrum) Serialize() []byte {
    s.mux.Lock()
    defer s.mux.Unlock()
    resp, _ := json.Marshal(s)
    return resp
}

var left [InputBufferSize]float64
var right [InputBufferSize]float64

var spec Spectrum

func parseBuffer(stream pulsesource.Source) {
    for {
        _ = stream.Read(left[:], right[:])
        ldata := fft.FFTReal(left[:])
        rdata := fft.FFTReal(right[:])
        n := len(ldata)
        var amps_left [7]int32
        var amps_right [7]int32
        for i, v := range ldata {
            amp := AmplitudeScale * cmplx.Abs(v) / float64(n)
            frequency := SampleRate * float64(i) / float64(n)
            band := getBand(frequency)
            amps_left[band] += int32(amp)
        }
        for i, v := range rdata {
            amp := AmplitudeScale * cmplx.Abs(v) / float64(n)
            frequency := SampleRate * float64(i) / float64(n)
            band := getBand(frequency)
            amps_right[band] += int32(math.Abs(amp))
        }
        _, max_left := MinMax(left[:])
        _, max_right := MinMax(right[:])
        spec.Set(amps_left[:], amps_right[:], max_left, max_right)
    }
}

func server(con net.Conn, stream pulsesource.Source) {
    for {
        buf := make([]byte, 512)
        _, _ = con.Read(buf)
        resp := spec.Serialize()
        _, _ = con.Write([]byte(resp))
    }
}

func main() {
    runtime.LockOSThread()

    os.Remove("/tmp/spectrum.sock")
    sock, _ := net.Listen("unix", "/tmp/spectrum.sock")
    stream, _ := pulsesource.New(SampleRate)
    fmt.Println("spectrum server online")

    go parseBuffer(stream)

    for {
        req, _ := sock.Accept()
        go server(req, stream)
    }
}
