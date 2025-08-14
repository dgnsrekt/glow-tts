package tts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// PCMFormat represents PCM audio format parameters
type PCMFormat struct {
	SampleRate   int
	Channels     int
	BitDepth     int
	ByteOrder    binary.ByteOrder
	IsSigned     bool
	IsFloat      bool
}

// DefaultPCMFormat returns the default PCM format for TTS
func DefaultPCMFormat() PCMFormat {
	return PCMFormat{
		SampleRate:   SampleRate,
		Channels:     Channels,
		BitDepth:     BitDepth,
		ByteOrder:    binary.LittleEndian,
		IsSigned:     true,
		IsFloat:      false,
	}
}

// BytesPerSample returns the number of bytes per sample
func (f PCMFormat) BytesPerSample() int {
	return f.BitDepth / 8 * f.Channels
}

// ValidatePCMData validates that PCM data matches the expected format
func ValidatePCMData(data []byte, format PCMFormat) error {
	if len(data) == 0 {
		return errors.New("empty PCM data")
	}
	
	bytesPerSample := format.BytesPerSample()
	if len(data)%bytesPerSample != 0 {
		return fmt.Errorf("PCM data length %d is not aligned to %d-byte samples", 
			len(data), bytesPerSample)
	}
	
	return nil
}

// CalculatePCMDuration calculates the duration of PCM audio data
func CalculatePCMDuration(dataLen int, format PCMFormat) float64 {
	if format.SampleRate == 0 || format.BytesPerSample() == 0 {
		return 0
	}
	
	numSamples := dataLen / format.BytesPerSample()
	return float64(numSamples) / float64(format.SampleRate)
}

// GenerateSilence generates silent PCM data for the given duration
func GenerateSilence(durationSeconds float64, format PCMFormat) []byte {
	numSamples := int(durationSeconds * float64(format.SampleRate))
	dataLen := numSamples * format.BytesPerSample()
	return make([]byte, dataLen)
}

// PCMReader provides utilities for reading PCM data
type PCMReader struct {
	reader io.Reader
	format PCMFormat
	buffer []byte
}

// NewPCMReader creates a new PCM reader
func NewPCMReader(reader io.Reader, format PCMFormat) *PCMReader {
	return &PCMReader{
		reader: reader,
		format: format,
		buffer: make([]byte, format.BytesPerSample()),
	}
}

// ReadSample reads a single PCM sample
func (r *PCMReader) ReadSample() ([]int16, error) {
	n, err := io.ReadFull(r.reader, r.buffer)
	if err != nil {
		return nil, err
	}
	
	if n != len(r.buffer) {
		return nil, io.ErrUnexpectedEOF
	}
	
	samples := make([]int16, r.format.Channels)
	buf := bytes.NewReader(r.buffer)
	
	for i := 0; i < r.format.Channels; i++ {
		if r.format.BitDepth == 16 {
			var sample int16
			err := binary.Read(buf, r.format.ByteOrder, &sample)
			if err != nil {
				return nil, err
			}
			samples[i] = sample
		} else if r.format.BitDepth == 8 {
			var sample int8
			err := binary.Read(buf, r.format.ByteOrder, &sample)
			if err != nil {
				return nil, err
			}
			// Convert 8-bit to 16-bit
			samples[i] = int16(sample) << 8
		}
	}
	
	return samples, nil
}

// PCMWriter provides utilities for writing PCM data
type PCMWriter struct {
	writer io.Writer
	format PCMFormat
	buffer *bytes.Buffer
}

// NewPCMWriter creates a new PCM writer
func NewPCMWriter(writer io.Writer, format PCMFormat) *PCMWriter {
	return &PCMWriter{
		writer: writer,
		format: format,
		buffer: new(bytes.Buffer),
	}
}

// WriteSample writes a single PCM sample
func (w *PCMWriter) WriteSample(samples []int16) error {
	if len(samples) != w.format.Channels {
		return fmt.Errorf("expected %d channels, got %d", w.format.Channels, len(samples))
	}
	
	w.buffer.Reset()
	
	for _, sample := range samples {
		if w.format.BitDepth == 16 {
			err := binary.Write(w.buffer, w.format.ByteOrder, sample)
			if err != nil {
				return err
			}
		} else if w.format.BitDepth == 8 {
			// Convert 16-bit to 8-bit
			sample8 := int8(sample >> 8)
			err := binary.Write(w.buffer, w.format.ByteOrder, sample8)
			if err != nil {
				return err
			}
		}
	}
	
	_, err := w.writer.Write(w.buffer.Bytes())
	return err
}

// ResamplePCM performs simple linear resampling of PCM data
// This is a basic implementation suitable for TTS quality requirements
func ResamplePCM(input []byte, inputFormat, outputFormat PCMFormat) ([]byte, error) {
	if inputFormat.Channels != outputFormat.Channels {
		return nil, errors.New("channel count conversion not supported")
	}
	
	if inputFormat.BitDepth != outputFormat.BitDepth {
		return nil, errors.New("bit depth conversion not supported")
	}
	
	if inputFormat.SampleRate == outputFormat.SampleRate {
		// No resampling needed
		return input, nil
	}
	
	// Calculate resampling ratio
	ratio := float64(outputFormat.SampleRate) / float64(inputFormat.SampleRate)
	
	// Calculate output size
	inputSamples := len(input) / inputFormat.BytesPerSample()
	outputSamples := int(float64(inputSamples) * ratio)
	output := make([]byte, outputSamples*outputFormat.BytesPerSample())
	
	// Simple linear interpolation
	inputReader := bytes.NewReader(input)
	outputBuffer := bytes.NewBuffer(output[:0])
	
	pcmReader := NewPCMReader(inputReader, inputFormat)
	pcmWriter := NewPCMWriter(outputBuffer, outputFormat)
	
	// Read all input samples
	var inputSampleData [][]int16
	for {
		sample, err := pcmReader.ReadSample()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		inputSampleData = append(inputSampleData, sample)
	}
	
	// Generate output samples with linear interpolation
	for i := 0; i < outputSamples; i++ {
		// Calculate corresponding position in input
		inputPos := float64(i) / ratio
		inputIdx := int(inputPos)
		fraction := inputPos - float64(inputIdx)
		
		if inputIdx >= len(inputSampleData)-1 {
			// Use last sample
			if len(inputSampleData) > 0 {
				pcmWriter.WriteSample(inputSampleData[len(inputSampleData)-1])
			}
		} else {
			// Linear interpolation between two samples
			sample1 := inputSampleData[inputIdx]
			sample2 := inputSampleData[inputIdx+1]
			
			interpolated := make([]int16, len(sample1))
			for ch := 0; ch < len(sample1); ch++ {
				val := float64(sample1[ch])*(1-fraction) + float64(sample2[ch])*fraction
				interpolated[ch] = int16(val)
			}
			
			pcmWriter.WriteSample(interpolated)
		}
	}
	
	return outputBuffer.Bytes(), nil
}

// NormalizePCMVolume normalizes the volume of PCM audio data
func NormalizePCMVolume(data []byte, format PCMFormat, targetLevel float64) ([]byte, error) {
	if targetLevel < 0 || targetLevel > 1 {
		return nil, errors.New("target level must be between 0 and 1")
	}
	
	if format.BitDepth != 16 {
		return nil, errors.New("only 16-bit audio supported for normalization")
	}
	
	// Find peak amplitude
	var peak int16
	reader := bytes.NewReader(data)
	for {
		var sample int16
		err := binary.Read(reader, format.ByteOrder, &sample)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		if sample < 0 {
			sample = -sample
		}
		if sample > peak {
			peak = sample
		}
	}
	
	if peak == 0 {
		// Silent audio, no normalization needed
		return data, nil
	}
	
	// Calculate scaling factor
	maxValue := int16(32767)
	targetPeak := int16(float64(maxValue) * targetLevel)
	scale := float64(targetPeak) / float64(peak)
	
	// Apply scaling
	output := make([]byte, len(data))
	reader = bytes.NewReader(data)
	writer := bytes.NewBuffer(output[:0])
	
	for {
		var sample int16
		err := binary.Read(reader, format.ByteOrder, &sample)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		// Scale and clamp
		scaled := float64(sample) * scale
		if scaled > float64(maxValue) {
			sample = maxValue
		} else if scaled < float64(-maxValue) {
			sample = -maxValue
		} else {
			sample = int16(scaled)
		}
		
		binary.Write(writer, format.ByteOrder, sample)
	}
	
	return writer.Bytes(), nil
}

// MixPCM mixes two PCM audio streams
func MixPCM(data1, data2 []byte, format PCMFormat) ([]byte, error) {
	if len(data1) != len(data2) {
		return nil, errors.New("audio data must be the same length")
	}
	
	if format.BitDepth != 16 {
		return nil, errors.New("only 16-bit audio supported for mixing")
	}
	
	output := make([]byte, len(data1))
	
	reader1 := bytes.NewReader(data1)
	reader2 := bytes.NewReader(data2)
	writer := bytes.NewBuffer(output[:0])
	
	for {
		var sample1, sample2 int16
		
		err := binary.Read(reader1, format.ByteOrder, &sample1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		err = binary.Read(reader2, format.ByteOrder, &sample2)
		if err != nil {
			return nil, err
		}
		
		// Mix with clamping
		mixed := int32(sample1) + int32(sample2)
		if mixed > 32767 {
			mixed = 32767
		} else if mixed < -32768 {
			mixed = -32768
		}
		
		binary.Write(writer, format.ByteOrder, int16(mixed))
	}
	
	return writer.Bytes(), nil
}