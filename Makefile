include ./cgo-flags.env

# Remove quotes from CGO_CFLAGS and CGO_LDFLAGS
CFLAGS := $(shell echo $(CGO_CFLAGS) | sed "s/^'//;s/'$$//")
LDFLAGS := $(shell echo $(CGO_LDFLAGS) | sed "s/^'//;s/'$$//")

webrtc-ros-bridge-client: receiver/peer_connection_channel/libvp8decoder.so sender/peer_connection_channel/libvp8encoder.so rclgo_gen cgo-flags.env
	CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) go build -o wrb

receiver/peer_connection_channel/libvp8decoder.so: receiver/peer_connection_channel/vp8_decoder.c receiver/peer_connection_channel/vp8_decoder.h
	cd receiver/peer_connection_channel && gcc -shared -o libvp8decoder.so -fPIC vp8_decoder.c $(pkg-config --cflags --libs vpx) $(CFLAGS) $(LDFLAGS)

sender/peer_connection_channel/libvp8encoder.so: sender/peer_connection_channel/vp8_encoder.c sender/peer_connection_channel/vp8_encoder.h
	cd sender/peer_connection_channel && gcc -shared -o libvp8encoder.so -fPIC vp8_encoder.c $(pkg-config --cflags --libs vpx) $(CFLAGS) $(LDFLAGS)

receiver/ros_channel/msgs cgo-flags.env: scripts/filter-root-path.sh
	go run github.com/tiiuae/rclgo/cmd/rclgo-gen generate -d rclgo_gen --root-path=$(shell ./scripts/filter-root-path.sh)

.PHONY: clean
clean:
	rm -rf wrb peer_connection_channel/libvp8decoder.so ros_channel/msgs cgo-flags.env
