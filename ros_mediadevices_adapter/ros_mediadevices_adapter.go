package rosmediadevicesadapter

import (
	"fmt"
	"image"
	"io"
	"sync"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

type rosImageAdapter struct {
	mu        sync.RWMutex
	lastFrame *image.RGBA
	doneCh    chan struct{}
}

func init() {
	Initialize()
}

func Initialize() {
	adapter := newROSImageAdapter()
	driver.GetManager().Register(adapter, driver.Info{
		Label:      "ros_image_topic",
		DeviceType: driver.Camera, // or driver.Screen depending on your use case
		Priority:   driver.PriorityNormal,
	})
}

func newROSImageAdapter() *rosImageAdapter {
	return &rosImageAdapter{}
}

func (a *rosImageAdapter) Open() error {
	a.doneCh = make(chan struct{})
	return nil
}

func (a *rosImageAdapter) Close() error {
	close(a.doneCh)
	return nil
}

func (a *rosImageAdapter) UpdateFrame(rosImg *sensor_msgs_msg.Image) error {
	rgba, err := ROSImageToRGBA(rosImg)
	if err != nil {
		return fmt.Errorf("failed to convert ROS image: %v", err)
	}

	a.mu.Lock()
	a.lastFrame = rgba
	a.mu.Unlock()
	return nil
}

func (a *rosImageAdapter) VideoRecord(selectedProp prop.Media) (video.Reader, error) {
	r := video.ReaderFunc(func() (img image.Image, release func(), err error) {
		select {
		case <-a.doneCh:
			return nil, nil, io.EOF
		default:
		}

		a.mu.RLock()
		if a.lastFrame == nil {
			a.mu.RUnlock()
			return nil, nil, fmt.Errorf("no frame available")
		}
		frame := a.lastFrame
		a.mu.RUnlock()

		return frame, func() {}, nil
	})
	return r, nil
}

func (a *rosImageAdapter) Properties() []prop.Media {
	// You might want to make these configurable or get from the ROS topic
	supportedProp := prop.Media{
		Video: prop.Video{
			Width:       640, // Default width
			Height:      480, // Default height
			FrameFormat: frame.FormatRGBA,
		},
	}
	return []prop.Media{supportedProp}
}

// ROSImageToRGBA converts sensor_msgs/Image to image.RGBA
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
