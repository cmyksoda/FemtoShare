package transcode

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mtlynch/picoshare/picoshare"
)

type (
	probeStream struct {
		CodecType     string `json:"codec_type"`
		CodecName     string `json:"codec_name"`
		PixFmt        string `json:"pix_fmt"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
		Level         int    `json:"level"`
		ColorTransfer string `json:"color_transfer"`
		Disposition   struct {
			AttachedPic int `json:"attached_pic"`
		} `json:"disposition"`
	}

	videoProbe struct {
		Streams []probeStream `json:"streams"`
	}
)

// Many hardware decoders reject H.264 above level 4.2 or frames larger than
// 1080p, which is why tall phone screen recordings fail in some browsers.
const (
	maxDimension = 1920
	maxH264Level = 42
)

func probeVideo(path string) (videoProbe, error) {
	output, err := exec.Command("ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		path).Output()
	if err != nil {
		return videoProbe{}, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probe videoProbe
	if err := json.Unmarshal(output, &probe); err != nil {
		return videoProbe{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}
	return probe, nil
}

func (p videoProbe) videoStream() (probeStream, bool) {
	for _, s := range p.Streams {
		if s.CodecType == "video" && s.Disposition.AttachedPic == 0 {
			return s, true
		}
	}
	return probeStream{}, false
}

func (p videoProbe) needsConversion(contentType picoshare.ContentType) bool {
	video, ok := p.videoStream()
	if !ok {
		return false
	}

	if contentType != outputContentType {
		return true
	}
	if video.CodecName != "h264" || (video.PixFmt != "yuv420p" && video.PixFmt != "yuvj420p") {
		return true
	}
	if isHDR(video.ColorTransfer) {
		return true
	}
	if max(video.Width, video.Height) > maxDimension || video.Level > maxH264Level {
		return true
	}

	for _, s := range p.Streams {
		if s.CodecType == "audio" && s.CodecName != "aac" && s.CodecName != "mp3" {
			return true
		}
	}
	return false
}

func (p videoProbe) ffmpegArgs(inputPath, outputPath string) []string {
	video, _ := p.videoStream()

	var filters []string
	if isHDR(video.ColorTransfer) {
		// Tone map HDR down to SDR so it doesn't look washed out on regular screens.
		filters = append(filters,
			fmt.Sprintf("zscale=tin=%s:pin=bt2020:min=bt2020nc:t=linear:npl=100", video.ColorTransfer),
			"format=gbrpf32le",
			"zscale=p=bt709",
			"tonemap=tonemap=hable:desat=0",
			"zscale=t=bt709:m=bt709:r=tv")
	}
	filters = append(filters,
		fmt.Sprintf(`scale=w=trunc(iw*min(1\,%d/max(iw\,ih))/2)*2:h=-2`, maxDimension),
		"format=yuv420p")

	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a:0?",
		// Drops embedded metadata such as the GPS location phones record.
		"-map_metadata", "-1",
		"-vf", strings.Join(filters, ","),
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-profile:v", "high",
		"-level:v", "4.2",
		"-fpsmax", "60",
		"-fps_mode", "cfr",
		"-color_primaries", "bt709",
		"-color_trc", "bt709",
		"-colorspace", "bt709",
		"-c:a", "aac",
		"-b:a", "160k",
		"-ac", "2",
		"-movflags", "+faststart",
		outputPath,
	}
}

func isHDR(colorTransfer string) bool {
	return colorTransfer == "arib-std-b67" || colorTransfer == "smpte2084"
}
