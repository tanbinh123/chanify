package core

import (
	"bytes"
	"encoding/json"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chanify/chanify/crypto"
	"github.com/chanify/chanify/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

const (
	pngHeader  = "\x89PNG\r\n\x1a\n"
	gifHeader  = "GIF"
	riffHeader = "RIFF"
	webpHeader = "WEBP"
)

func (c *Core) bindBodyJSON(ctx *gin.Context, obj interface{}) error {
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
	}
	ctx.Set(gin.BodyBytesKey, body)
	if strings.HasPrefix(ctx.ContentType(), "application/x-chsec-json") {
		body, err = c.logic.Decrypt(body)
		if err != nil {
			return err
		}
	}
	return json.Unmarshal(body, obj)
}

func verifyUser(ctx *gin.Context, key string) bool {
	sign, err := crypto.Base64Encode.DecodeString(ctx.GetHeader("CHUserSign"))
	if err != nil {
		return false
	}
	data, _ := ctx.Get(gin.BodyBytesKey)
	return verifySign(key, sign, data.([]byte))
}

func verifyDevice(ctx *gin.Context, key string) bool {
	sign, err := crypto.Base64Encode.DecodeString(ctx.GetHeader("CHDevSign"))
	if err != nil {
		return false
	}
	data, _ := ctx.Get(gin.BodyBytesKey)
	return verifySign(key, sign, data.([]byte))
}

func verifySign(key string, sign []byte, data []byte) bool {
	kd, err := crypto.Base64Encode.DecodeString(key)
	if err != nil {
		return false
	}
	pk, err := crypto.LoadPublicKey(kd)
	if err != nil {
		return false
	}
	return pk.Verify(data, sign)
}

func (c *Core) parseToken(token string) (*model.Token, error) {
	tk, err := model.ParseToken(token)
	if err != nil {
		return nil, err
	}
	if !c.logic.VerifyToken(tk) {
		return nil, model.ErrInvalidToken
	}
	return tk, nil
}

func getToken(ctx *gin.Context) string {
	token := ctx.GetHeader("token")
	if len(token) <= 0 {
		token = ctx.Query("token")
		if len(token) <= 0 {
			token = ctx.Param("token")
			if len(token) > 0 && token[0] == '/' {
				token = token[1:]
			}
		}
	}
	return token
}

func parsePriority(priority string) int {
	if len(priority) > 0 {
		if p, err := strconv.Atoi(priority); err == nil {
			return p
		}
	}
	return 0
}

func parseImageContentType(data []byte) string {
	if len(data) > 12 {
		str := string(data[:12])
		if strings.HasPrefix(str, pngHeader) {
			return "image/png"
		} else if strings.HasPrefix(str, gifHeader) {
			return "image/gif"
		} else if strings.HasPrefix(str, "\x49\x49") || strings.HasPrefix(str, "\x4D\x4D") {
			return "image/tiff"
		} else if strings.HasPrefix(str, riffHeader) && strings.HasPrefix(string(str[8:]), webpHeader) {
			return "image/webp"
		}
	}
	return "image/jpeg"
}

func createThumbnail(data []byte) *model.Thumbnail {
	switch parseImageContentType(data) {
	case "image/png":
		if cfg, err := png.DecodeConfig(bytes.NewReader(data)); err == nil {
			return model.NewThumbnail(cfg.Width, cfg.Height)
		}
	case "image/gif":
		if cfg, err := gif.DecodeConfig(bytes.NewReader(data)); err == nil {
			return model.NewThumbnail(cfg.Width, cfg.Height)
		}
	case "image/tiff":
		if cfg, err := tiff.DecodeConfig(bytes.NewReader(data)); err == nil {
			return model.NewThumbnail(cfg.Width, cfg.Height)
		}
	case "image/webp":
		if cfg, err := webp.DecodeConfig(bytes.NewReader(data)); err == nil {
			return model.NewThumbnail(cfg.Width, cfg.Height)
		}
	default:
		if cfg, err := jpeg.DecodeConfig(bytes.NewReader(data)); err == nil {
			return model.NewThumbnail(cfg.Width, cfg.Height)
		}
	}
	return nil
}

func fileBaseName(path string) string {
	name := ""
	if len(path) > 0 {
		_, fname := filepath.Split(path)
		if len(fname) > 0 && fname[0] != '.' {
			name = fname
		}
	}
	return name
}

func fixLog(s string) string {
	return strings.Replace(strings.Replace(s, "\n", "", -1), "\r", "", -1)
}

// JSONString define boolean string
type JSONString string

// UnmarshalJSON for boolean string
func (s *JSONString) UnmarshalJSON(data []byte) error {
	asString := strings.Trim(string(data), "\"")
	switch asString {
	case "1", "true", "TRUE", "True", "On", "on":
		*s = "1"
	case "0", "false", "FALSE", "False", "Off", "off", "none", "NONE", "null", "NULL":
		*s = ""
	default:
		*s = JSONString(asString)
	}
	return nil
}
