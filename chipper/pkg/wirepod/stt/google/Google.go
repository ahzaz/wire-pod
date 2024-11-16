package google

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	sr "github.com/kercre123/wire-pod/chipper/pkg/wirepod/speechrequest"
	"google.golang.org/api/option"
)

var Name string = "google"

var client *speech.Client

var ctx context.Context

func Init() error {

	if os.Getenv("GOOGLE_STT_API_KEY") == "" {
		logger.Println("Google STT API Key not found.")
		return fmt.Errorf("google stt api key not found")
	}

	apiKey := os.Getenv("GOOGLE_STT_API_KEY")
	ctx = context.Background()
	c, err := speech.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return err
	}
	client = c
	logger.Println("Google client for speech-to-text initialized!")
	return nil
}

func STT(sreq sr.SpeechRequest) (string, error) {
	logger.Println("Incoming request")
	var err error
	rp, wp := io.Pipe()

	done := make(chan bool)
	speechDone := false
	go func(wp *io.PipeWriter) {
		defer wp.Close()

		for {
			select {
			case <-done:
				return
			default:
				var chunk []byte
				chunk, err = sreq.GetNextStreamChunk()
				speechDone, _ = sreq.DetectEndOfSpeech()
				if err != nil {
					fmt.Println("End of stream")
					return
				}
				wp.Write(chunk)
				if speechDone {
					return
				}
			}
		}
	}(wp)

	data := new(bytes.Buffer)
	if _, err := io.Copy(data, rp); err != nil {
		panic(err)
	}

	resp, _ := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data.Bytes()},
		},
	})

	var transcript = ""
	var maxScore float32
	maxScore = 0

	if resp == nil {
		logger.Println("nil response for STT")
		return "", errors.New("nil response for STT")
	}

	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			logger.Println("\"%s\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
			if alt.Confidence > maxScore {
				transcript = alt.Transcript
				maxScore = alt.Confidence
			}
		}
	}
	return transcript, nil
}
