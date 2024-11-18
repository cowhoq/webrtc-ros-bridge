#ifndef VP8_ENCODER_H_
#define VP8_ENCODER_H_

#include <sensor_msgs/msg/image.h>
#include <vpx/vpx_codec.h>

int init_encoder(vpx_codec_ctx_t *codec, unsigned int w, unsigned int h,
                 int bitrate);
int set_bitrate(vpx_codec_ctx_t *codec, int bitrate);
void ros_to_vpx_image(const sensor_msgs__msg__Image *ros_img,
                      vpx_image_t *vpx_img);
int encode_frame(vpx_codec_ctx_t *codec, vpx_image_t *vpx_img, uint8_t **data,
                 size_t *data_size);
void cleanup_vpx_image(vpx_image_t *vpx_img);

#endif // !VP8_ENCODER_H_
