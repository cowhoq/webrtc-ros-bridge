#ifndef VP8_DECODER_H_
#define VP8_DECODER_H_

#include <sensor_msgs/msg/image.h>
#include <vpx/vp8.h>
#include <vpx/vp8dx.h>
#include <vpx/vpx_decoder.h>

int init_decoder(vpx_codec_ctx_t *codec, unsigned int w, unsigned int h);
int decode_frame(vpx_codec_ctx_t *codec, const uint8_t *data, size_t data_size);
vpx_image_t *get_frame(vpx_codec_ctx_t *codec);
void vpx_to_ros_image(const vpx_image_t *vpx_img,
                      sensor_msgs__msg__Image *ros_img);
void cleanup_ros_image(sensor_msgs__msg__Image *ros_img);

#endif // VP8_DECODER_H_
