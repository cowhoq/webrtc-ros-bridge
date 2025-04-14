// Package bandwidthmanager implements adaptive bandwidth allocation strategies
// to optimize WebRTC performance for ROS message transmission, prioritizing
// different message types based on their importance and current network conditions.
package bandwidthmanager

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Priority levels for different types of messages
const (
	PriorityLow    = 1
	PriorityMedium = 2
	PriorityHigh   = 3
	PriorityVideo  = 4 // Video always has highest priority
)

// BandwidthManagerConfig holds the configuration for the bandwidth manager
type BandwidthManagerConfig struct {
	// TotalBandwidth is the maximum available bandwidth in bps
	TotalBandwidth int

	// MinVideoBitrate is the minimum bitrate for video streams in bps
	MinVideoBitrate int

	// MaxVideoBitrate is the maximum bitrate for video streams in bps
	MaxVideoBitrate int

	// TargetVideoBitrate is the desired video bitrate under optimal conditions
	TargetVideoBitrate int

	// MinDataChannelBandwidth is the minimum bandwidth reserved for data channels
	MinDataChannelBandwidth int

	// AdjustInterval defines how often the bandwidth allocation is recalculated
	AdjustInterval time.Duration

	// QualityAdjustSensitivity controls how aggressively bitrate changes are applied (0-1)
	QualityAdjustSensitivity float64
}

// VP8Encoder接口定义了VP8编码器需要提供的方法
type VP8Encoder interface {
	SetBitRate(bitRate int) error
}

// BandwidthManager dynamically allocates bandwidth between video and data channels
// to provide optimal quality of service based on current usage patterns
type BandwidthManager struct {
	config BandwidthManagerConfig

	// Current video bitrate
	currentVideoBitrate int

	// Current bandwidth used by data channels
	currentDataBandwidth int

	// Statistics on data traffic by message type
	dataTrafficStats map[string]int

	// Mutex for thread safety
	mu sync.Mutex

	// Video encoder reference
	videoEncoder interface{}

	// Data channel reference
	dataChannel *webrtc.DataChannel

	// Channel to signal shutdown
	done chan struct{}
}

// NewBandwidthManager creates a new bandwidth manager with the provided configuration
func NewBandwidthManager(config BandwidthManagerConfig) *BandwidthManager {
	// Apply default values if not provided
	if config.TotalBandwidth == 0 {
		config.TotalBandwidth = 10_000_000 // 10 Mbps default
	}
	if config.MinVideoBitrate == 0 {
		config.MinVideoBitrate = 500_000 // 500 Kbps minimum video quality
	}
	if config.MaxVideoBitrate == 0 {
		config.MaxVideoBitrate = 8_000_000 // 8 Mbps maximum video quality
	}
	if config.TargetVideoBitrate == 0 {
		config.TargetVideoBitrate = 5_000_000 // 5 Mbps target video bitrate
	}
	if config.MinDataChannelBandwidth == 0 {
		config.MinDataChannelBandwidth = 500_000 // 500 Kbps minimum data channel bandwidth
	}
	if config.AdjustInterval == 0 {
		config.AdjustInterval = 2 * time.Second // Adjust every 2 seconds
	}
	if config.QualityAdjustSensitivity == 0 {
		config.QualityAdjustSensitivity = 0.3 // Default sensitivity
	}

	return &BandwidthManager{
		config:               config,
		currentVideoBitrate:  config.TargetVideoBitrate,
		currentDataBandwidth: config.TotalBandwidth - config.TargetVideoBitrate,
		dataTrafficStats:     make(map[string]int),
		done:                 make(chan struct{}),
	}
}

// SetVideoEncoder registers the video encoder with the bandwidth manager
func (bm *BandwidthManager) SetVideoEncoder(encoder interface{}) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.videoEncoder = encoder
}

// SetDataChannel registers the data channel with the bandwidth manager
func (bm *BandwidthManager) SetDataChannel(dc *webrtc.DataChannel) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.dataChannel = dc
}

// RegisterMessageTraffic records the traffic for a specific message type
func (bm *BandwidthManager) RegisterMessageTraffic(msgType string, bytes int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Update traffic statistics for the specific message type
	if _, exists := bm.dataTrafficStats[msgType]; !exists {
		bm.dataTrafficStats[msgType] = 0
	}
	bm.dataTrafficStats[msgType] += bytes
}

// GetCurrentVideoBitrate returns the current video bitrate setting
func (bm *BandwidthManager) GetCurrentVideoBitrate() int {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.currentVideoBitrate
}

// Start begins the bandwidth management process
func (bm *BandwidthManager) Start() {
	go bm.adjustBandwidthLoop()
}

// Stop terminates the bandwidth management process
func (bm *BandwidthManager) Stop() {
	close(bm.done)
}

// adjustBandwidthLoop periodically adjusts bandwidth allocation
func (bm *BandwidthManager) adjustBandwidthLoop() {
	ticker := time.NewTicker(bm.config.AdjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.adjustBandwidth()
		case <-bm.done:
			return
		}
	}
}

// adjustBandwidth recalculates bandwidth allocation based on current usage
func (bm *BandwidthManager) adjustBandwidth() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Calculate total data channel traffic
	totalDataTraffic := 0
	for _, bytes := range bm.dataTrafficStats {
		totalDataTraffic += bytes
	}

	// Convert bytes to bits per second
	dataBandwidthUsage := int(float64(totalDataTraffic) * 8 / bm.config.AdjustInterval.Seconds())

	// Reset statistics
	bm.dataTrafficStats = make(map[string]int)

	// Calculate how much bandwidth should be allocated to video
	// If data channels are using less than allocated, give excess to video
	remainingBandwidth := bm.config.TotalBandwidth - dataBandwidthUsage

	// Ensure data channels get at least their minimum guaranteed bandwidth
	if remainingBandwidth < bm.config.MinVideoBitrate {
		remainingBandwidth = bm.config.MinVideoBitrate
	}

	// Calculate new video bitrate with upper bound
	targetVideoBitrate := remainingBandwidth
	if targetVideoBitrate > bm.config.MaxVideoBitrate {
		targetVideoBitrate = bm.config.MaxVideoBitrate
	}

	// Apply smooth adjustment to avoid quality fluctuations
	smoothFactor := bm.config.QualityAdjustSensitivity
	newVideoBitrate := int(float64(bm.currentVideoBitrate)*(1-smoothFactor) + float64(targetVideoBitrate)*smoothFactor)

	// Ensure bitrate stays within configured bounds
	if newVideoBitrate < bm.config.MinVideoBitrate {
		newVideoBitrate = bm.config.MinVideoBitrate
	} else if newVideoBitrate > bm.config.MaxVideoBitrate {
		newVideoBitrate = bm.config.MaxVideoBitrate
	}

	// Only update if change is significant enough
	thresholdPct := 0.05 // 5% change threshold
	if float64(abs(newVideoBitrate-bm.currentVideoBitrate))/float64(bm.currentVideoBitrate) > thresholdPct {
		// Update video bitrate
		bm.currentVideoBitrate = newVideoBitrate

		// Log the adjustment
		slog.Info("Bandwidth allocation adjusted",
			"videoBitrate", newVideoBitrate,
			"dataUsage", dataBandwidthUsage,
			"totalBandwidth", bm.config.TotalBandwidth)

		// 应用比特率调整到编码器
		if err := bm.UpdateEncoderBitrate(newVideoBitrate); err != nil {
			slog.Error("Failed to update encoder bitrate", "error", err)
		}
	}
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// SetInitialVideoBitrate allows setting the initial video bitrate
func (bm *BandwidthManager) SetInitialVideoBitrate(bitrate int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.currentVideoBitrate = bitrate
}

// UpdateEncoderBitrate尝试更新编码器的比特率
func (bm *BandwidthManager) UpdateEncoderBitrate(bitrate int) error {
	if bm.videoEncoder == nil {
		return nil // 没有注册编码器，忽略
	}

	// 尝试将编码器转换为VP8Encoder接口
	if vpxEncoder, ok := bm.videoEncoder.(VP8Encoder); ok {
		return vpxEncoder.SetBitRate(bitrate)
	}

	return nil // 不支持的编码器类型，忽略
}
