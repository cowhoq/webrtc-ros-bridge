include ./cgo-flags.env

# Remove quotes from CGO_CFLAGS and CGO_LDFLAGS
CFLAGS := $(shell echo $(CGO_CFLAGS) | sed "s/^'//;s/'$$//")
LDFLAGS := $(shell echo $(CGO_LDFLAGS) | sed "s/^'//;s/'$$//")

webrtc-ros-bridge-client: libvp8decoder.so ros_channel/msgs cgo-flags.env
	CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) go build

libvp8decoder.so: peer_connection_channel/vp8_decoder.c peer_connection_channel/vp8_decoder.h
	cd peer_connection_channel && gcc -shared -o libvp8decoder.so -fPIC vp8_decoder.c $(pkg-config --cflags --libs vpx) $(CFLAGS) $(LDFLAGS)

ros_channel/msgs cgo-flags.env: scripts/filter-root-path.sh
	go run github.com/tiiuae/rclgo/cmd/rclgo-gen generate -d ros_channel/msgs --include-go-package-deps ./... --root-path=$(shell ./scripts/filter-root-path.sh)

.PHONY: clean
clean:
	rm -rf webrtc-ros-bridge peer_connection_channel/libvp8decoder.so ros_channel/msgs cgo-flags.env
