webrtc-ros-bridge-client: libvp8decoder.so ros_channel/msgs
	go build

libvp8decoder.so: peer_connection_channel/vp8_decoder.c peer_connection_channel/vp8_decoder.h
	cd peer_connection_channel && gcc -shared -o libvp8decoder.so -fPIC vp8_decoder.c $(pkg-config --cflags --libs vpx)

ros_channel/msgs: generate.go
	go generate

.PHONY: clean
clean:
	rm -f webrtc-ros-bridge-client peer_connection_channel/libvp8decoder.so
