package transcode

import (
	"testing"

	"github.com/cmyksoda/femtoshare/femtoshare"
)

func TestNeedsConversion(t *testing.T) {
	browserFriendly := probeStream{
		CodecType: "video",
		CodecName: "h264",
		PixFmt:    "yuv420p",
		Width:     1920,
		Height:    1080,
		Level:     42,
	}
	aac := probeStream{CodecType: "audio", CodecName: "aac"}

	for _, tt := range []struct {
		description string
		contentType femtoshare.ContentType
		streams     []probeStream
		want        bool
	}{
		{
			description: "leaves browser-friendly H.264 alone",
			contentType: "video/mp4",
			streams:     []probeStream{browserFriendly, aac},
			want:        false,
		},
		{
			description: "converts 10-bit HDR HEVC from phone cameras",
			contentType: "video/mp4",
			streams: []probeStream{{
				CodecType:     "video",
				CodecName:     "hevc",
				PixFmt:        "yuv420p10le",
				Width:         1920,
				Height:        1080,
				ColorTransfer: "arib-std-b67",
			}, aac},
			want: true,
		},
		{
			description: "converts tall screen recordings with a high H.264 level",
			contentType: "video/mp4",
			streams: []probeStream{{
				CodecType: "video",
				CodecName: "h264",
				PixFmt:    "yuv420p",
				Width:     1280,
				Height:    2856,
				Level:     50,
			}},
			want: true,
		},
		{
			description: "converts QuickTime containers",
			contentType: "video/quicktime",
			streams:     []probeStream{browserFriendly, aac},
			want:        true,
		},
		{
			description: "converts audio that browsers can't play in MP4",
			contentType: "video/mp4",
			streams:     []probeStream{browserFriendly, {CodecType: "audio", CodecName: "pcm_s16le"}},
			want:        true,
		},
		{
			description: "ignores files without a video stream",
			contentType: "video/quicktime",
			streams:     []probeStream{aac},
			want:        false,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			probe := videoProbe{Streams: tt.streams}
			if got, want := probe.needsConversion(tt.contentType), tt.want; got != want {
				t.Errorf("needsConversion=%v, want=%v", got, want)
			}
		})
	}
}

func TestMp4Filename(t *testing.T) {
	for _, tt := range []struct {
		input femtoshare.Filename
		want  femtoshare.Filename
	}{
		{"clip.mov", "clip.mp4"},
		{"PXL_20261010_231108164.mp4", "PXL_20261010_231108164.mp4"},
		{"no-extension", "no-extension.mp4"},
		{"my.trip.mkv", "my.trip.mp4"},
	} {
		t.Run(string(tt.input), func(t *testing.T) {
			if got, want := mp4Filename(tt.input), tt.want; got != want {
				t.Errorf("mp4Filename=%q, want=%q", got, want)
			}
		})
	}
}

func TestFfmpegArgsUseMaxH264Level(t *testing.T) {
	args := videoProbe{Streams: []probeStream{{CodecType: "video"}}}.ffmpegArgs("in", "out")
	for i, arg := range args {
		if arg == "-level:v" {
			if got, want := args[i+1], "4.2"; got != want {
				t.Errorf("-level:v=%q, want=%q", got, want)
			}
			return
		}
	}
	t.Error("ffmpeg args are missing -level:v")
}
