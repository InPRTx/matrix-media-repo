package r0

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/turt2live/matrix-media-repo/api"
	"github.com/turt2live/matrix-media-repo/config"
	"github.com/turt2live/matrix-media-repo/services/url_service"
	"github.com/turt2live/matrix-media-repo/util"
	"github.com/turt2live/matrix-media-repo/util/errs"
)

type MatrixOpenGraph struct {
	Url         string `json:"og:url,omitempty"`
	SiteName    string `json:"og:site_name,omitempty"`
	Type        string `json:"og:type,omitempty"`
	Description string `json:"og:description,omitempty"`
	Title       string `json:"og:title,omitempty"`
	ImageMxc    string `json:"og:image,omitempty"`
	ImageType   string `json:"og:image:type,omitempty"`
	ImageSize   int64  `json:"matrix:image:size,omitempty"`
	ImageWidth  int    `json:"og:image:width,omitempty"`
	ImageHeight int    `json:"og:image:height,omitempty"`
}

func PreviewUrl(r *http.Request, log *logrus.Entry, user api.UserInfo) interface{} {
	if !config.Get().UrlPreviews.Enabled {
		return api.NotFoundError()
	}

	params := r.URL.Query()

	// Parse the parameters
	urlStr := params.Get("url")
	tsStr := params.Get("ts")
	ts := util.NowMillis()
	var err error
	if tsStr != "" {
		ts, err = strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			log.Error("Error parsing ts: " + err.Error())
			return api.BadRequest(err.Error())
		}
	}

	// Validate the URL
	if urlStr == "" {
		return api.BadRequest("No url provided")
	}
	if strings.Index(urlStr, "http://") != 0 && strings.Index(urlStr, "https://") != 0 {
		return api.BadRequest("Scheme not accepted")
	}

	svc := url_service.New(r.Context(), log)
	preview, err := svc.GetPreview(urlStr, r.Host, user.UserId, ts)
	if err != nil {
		if err == errs.ErrMediaNotFound || err == errs.ErrHostNotFound {
			return api.NotFoundError()
		} else if err == errs.ErrInvalidHost || err == errs.ErrHostBlacklisted {
			return api.BadRequest(err.Error())
		} else {
			return api.InternalServerError("unexpected error during request")
		}
	}

	return &MatrixOpenGraph{
		Url:         preview.Url,
		SiteName:    preview.SiteName,
		Type:        preview.Type,
		Description: preview.Description,
		Title:       preview.Title,
		ImageMxc:    preview.ImageMxc,
		ImageType:   preview.ImageType,
		ImageSize:   preview.ImageSize,
		ImageWidth:  preview.ImageWidth,
		ImageHeight: preview.ImageHeight,
	}
}