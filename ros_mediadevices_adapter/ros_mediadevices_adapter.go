package rosmediadevicesadapter

import (
	"fmt"
	"image"
	"io"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type rosImageAdapter struct {
	lastFrame *image.RGBA
	doneCh    chan struct{}
	imgChan   <-chan *sensor_msgs_msg.Image
	imgWidth  int
	imgHeight int
	frameRate float64
}

func Initialize(imgChan <-chan *sensor_msgs_msg.Image, width, height int, frameRate float64) {
	adapter := newROSImageAdapter(width, height, frameRate)
	adapter.imgChan = imgChan
	driver.GetManager().Register(adapter, driver.Info{
		Label:      "ros_image_topic",
		DeviceType: driver.Camera,
		Priority:   driver.PriorityHigh,
	})
}

func newROSImageAdapter(width, height int, frameRate float64) *rosImageAdapter {
	return &rosImageAdapter{imgWidth: width, imgHeight: height, frameRate: frameRate}
}

func (a *rosImageAdapter) getRgba() (*image.RGBA, error) {
	img := <-a.imgChan
	rgba, err := ROSImageToRGBA(img)
	if err != nil {
		return nil, err
	}
	return rgba, nil
}

func (a *rosImageAdapter) Open() error {
	a.doneCh = make(chan struct{})
	return nil
}

func (a *rosImageAdapter) Close() error {
	close(a.doneCh)
	return nil
}

func (a *rosImageAdapter) VideoRecord(selectedProp prop.Media) (video.Reader, error) {
	r := video.ReaderFunc(func() (img image.Image, release func(), err error) {
		select {
		case <-a.doneCh:
			return nil, nil, io.EOF
		default:
		}
		rgba, err := a.getRgba()
		if err != nil {
			return nil, func() {}, err
		}
		return rgba, func() {}, nil
	})
	return r, nil
}

func (a *rosImageAdapter) Properties() []prop.Media {
	supportedProp := prop.Media{
		Video: prop.Video{
			Width:       a.imgWidth,
			Height:      a.imgHeight,
			FrameFormat: frame.FormatRGBA,
			FrameRate:   float32(a.frameRate),
		},
	}
	return []prop.Media{supportedProp}
}

func ROSImageToRGBA(rosImg *sensor_msgs_msg.Image) (*image.RGBA, error) {
	width := int(rosImg.Width)
	height := int(rosImg.Height)
	stride := int(rosImg.Step)

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	switch rosImg.Encoding {
	case "rgb8":
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcIdx := y*stride + x*3
				dstIdx := y*rgba.Stride + x*4
				rgba.Pix[dstIdx+0] = rosImg.Data[srcIdx+0] // R
				rgba.Pix[dstIdx+1] = rosImg.Data[srcIdx+1] // G
				rgba.Pix[dstIdx+2] = rosImg.Data[srcIdx+2] // B
				rgba.Pix[dstIdx+3] = 255                   // A
			}
		}
	case "rgba8":
		copy(rgba.Pix, rosImg.Data)
	case "bgr8":
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				srcIdx := y*stride + x*3
				dstIdx := y*rgba.Stride + x*4
				rgba.Pix[dstIdx+0] = rosImg.Data[srcIdx+2] // R (from B)
				rgba.Pix[dstIdx+1] = rosImg.Data[srcIdx+1] // G
				rgba.Pix[dstIdx+2] = rosImg.Data[srcIdx+0] // B (from R)
				rgba.Pix[dstIdx+3] = 255                   // A
			}
		}
	default:
		return nil, fmt.Errorf("unsupported image encoding: %s", rosImg.Encoding)
	}

	return rgba, nil
}
