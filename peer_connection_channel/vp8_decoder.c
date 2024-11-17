// vp8_decoder.c
#include "vp8_decoder.h"
#include <rcutils/allocator.h>
#include <rosidl_runtime_c/primitives_sequence_functions.h>
#include <rosidl_runtime_c/string_functions.h>
#include <sensor_msgs/msg/image.h>
#include <string.h>
#include <vpx/vp8.h>
#include <vpx/vp8dx.h>
#include <vpx/vpx_decoder.h>
#include <vpx/vpx_image.h>

// Function to initialize the decoder
int init_decoder(vpx_codec_ctx_t *codec, unsigned int w, unsigned int h) {
  vpx_codec_dec_cfg_t cfg;
  cfg.w = w;
  cfg.h = h;
  return vpx_codec_dec_init(codec, vpx_codec_vp8_dx(), &cfg, 0);
}

// Function to decode VP8 frame
int decode_frame(vpx_codec_ctx_t *codec, const uint8_t *data,
                 size_t data_size) {
  return vpx_codec_decode(codec, data, data_size, NULL, 0);
}

// Function to get the decoded frame
vpx_image_t *get_frame(vpx_codec_ctx_t *codec) {
  vpx_codec_iter_t iter = NULL;
  return vpx_codec_get_frame(codec, &iter);
}

void vpx_to_ros_image(const vpx_image_t *vpx_img,
                      sensor_msgs__msg__Image *ros_img) {
  // Initialize ROS message
  sensor_msgs__msg__Image__init(ros_img);

  // Set image dimensions
  ros_img->width = vpx_img->d_w;
  ros_img->height = vpx_img->d_h;

  // Set encoding to bgr8 since we'll convert to BGR
  rosidl_runtime_c__String__init(&ros_img->encoding);
  rosidl_runtime_c__String__assign(&ros_img->encoding, "bgr8");

  // Set step (3 bytes per pixel for BGR)
  ros_img->step = ros_img->width * 3;

  // Allocate data array
  size_t data_size = ros_img->step * ros_img->height;
  rosidl_runtime_c__uint8__Sequence *seq = &ros_img->data;
  seq->data = (uint8_t *)malloc(data_size * sizeof(uint8_t));
  seq->size = data_size;
  seq->capacity = data_size;

  // Convert and copy data
  for (int y = 0; y < ros_img->height; y++) {
    for (int x = 0; x < ros_img->width; x++) {
      // Calculate correct indices using strides
      int y_idx = y * vpx_img->stride[VPX_PLANE_Y] + x;
      int u_idx = (y >> 1) * vpx_img->stride[VPX_PLANE_U] + (x >> 1);
      int v_idx = (y >> 1) * vpx_img->stride[VPX_PLANE_V] + (x >> 1);

      // Get YUV values
      int Y = vpx_img->planes[VPX_PLANE_Y][y_idx];
      int U = vpx_img->planes[VPX_PLANE_U][u_idx] - 128;
      int V = vpx_img->planes[VPX_PLANE_V][v_idx] - 128;

      // YUV to RGB conversion
      int R = Y + (1.403 * V);
      int G = Y - (0.344 * U) - (0.714 * V);
      int B = Y + (1.770 * U);

      // Clamp values to [0, 255]
      R = R < 0 ? 0 : (R > 255 ? 255 : R);
      G = G < 0 ? 0 : (G > 255 ? 255 : G);
      B = B < 0 ? 0 : (B > 255 ? 255 : B);

      // Write to destination in BGR order
      int dest_idx = (y * ros_img->width + x) * 3;
      seq->data[dest_idx + 0] = (unsigned char)R;
      seq->data[dest_idx + 1] = (unsigned char)G;
      seq->data[dest_idx + 2] = (unsigned char)B;
    }
  }
}

// Don't forget to clean up when done
void cleanup_ros_image(sensor_msgs__msg__Image *ros_img) {
  sensor_msgs__msg__Image__fini(ros_img);
}
