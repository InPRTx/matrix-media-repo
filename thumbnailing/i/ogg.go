package i

import (
	"errors"

	"github.com/faiface/beep/vorbis"
	"github.com/turt2live/matrix-media-repo/common/rcontext"
	"github.com/turt2live/matrix-media-repo/thumbnailing/m"
	"github.com/turt2live/matrix-media-repo/util"
)

type oggGenerator struct {
}

func (d oggGenerator) supportedContentTypes() []string {
	return []string{"audio/ogg"}
}

func (d oggGenerator) supportsAnimation() bool {
	return false
}

func (d oggGenerator) matches(img []byte, contentType string) bool {
	return contentType == "audio/ogg"
}

func (d oggGenerator) GenerateThumbnail(b []byte, contentType string, width int, height int, method string, animated bool, ctx rcontext.RequestContext) (*m.Thumbnail, error) {
	audio, format, err := vorbis.Decode(util.ByteCloser(b))
	if err != nil {
		return nil, errors.New("ogg: error decoding audio: " + err.Error())
	}

	defer audio.Close()
	return mp3Generator{}.GenerateFromStream(audio, format, width, height)
}

func init() {
	generators = append(generators, oggGenerator{})
}