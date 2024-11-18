// vp8_encoder.c
#include "vp8_encoder.h"
#include <rcutils/allocator.h>
#include <rosidl_runtime_c/primitives_sequence_functions.h>
#include <rosidl_runtime_c/string_functions.h>
#include <sensor_msgs/msg/image.h>
#include <string.h>
#include <vpx/vp8.h>
#include <vpx/vp8cx.h>
#include <vpx/vpx_encoder.h>
#include <vpx/vpx_image.h>

// Function to initialize the encoder
int init_encoder(vpx_codec_ctx_t *codec, unsigned int w, unsigned int h,
                 int bitrate) {
  vpx_codec_enc_cfg_t cfg;
  vpx_codec_err_t res;

  res = vpx_codec_enc_config_default(vpx_codec_vp8_cx(), &cfg, 0);
  if (res) {
    return res;
  }

  cfg.g_w = w;
  cfg.g_h = h;
  cfg.rc_target_bitrate = bitrate;

  return vpx_codec_enc_init(codec, vpx_codec_vp8_cx(), &cfg, 0);
}

// Function to set the bitrate dynamically
int set_bitrate(vpx_codec_ctx_t *codec, int bitrate) {
  return vpx_codec_control(codec, VP8E_SET_CQ_LEVEL, bitrate);
}

// Function to convert ROS image message to VPX image
void ros_to_vpx_image(const sensor_msgs__msg__Image *ros_img,
                      vpx_image_t *vpx_img) {
  vpx_img->fmt = VPX_IMG_FMT_I420;
  vpx_img->d_w = ros_img->width;
  vpx_img->d_h = ros_img->height;
  vpx_img->stride[VPX_PLANE_Y] = ros_img->width;
  vpx_img->stride[VPX_PLANE_U] = ros_img->width / 2;
  vpx_img->stride[VPX_PLANE_V] = ros_img->width / 2;

  vpx_img->planes[VPX_PLANE_Y] =
      (uint8_t *)malloc(ros_img->width * ros_img->height);
  vpx_img->planes[VPX_PLANE_U] =
      (uint8_t *)malloc(ros_img->width * ros_img->height / 4);
  vpx_img->planes[VPX_PLANE_V] =
      (uint8_t *)malloc(ros_img->width * ros_img->height / 4);

  for (int y = 0; y < ros_img->height; y++) {
    for (int x = 0; x < ros_img->width; x++) {
      int dest_idx = (y * ros_img->width + x) * 3;
      int R = ros_img->data.data[dest_idx + 0];
      int G = ros_img->data.data[dest_idx + 1];
      int B = ros_img->data.data[dest_idx + 2];

      // RGB to YUV conversion
      int Y = (0.299 * R + 0.587 * G + 0.114 * B);
      int U = (-0.169 * R - 0.331 * G + 0.500 * B) + 128;
      int V = (0.500 * R - 0.419 * G - 0.081 * B) + 128;

      vpx_img->planes[VPX_PLANE_Y][y * vpx_img->stride[VPX_PLANE_Y] + x] = Y;
      if (y % 2 == 0 && x % 2 == 0) {
        vpx_img->planes[VPX_PLANE_U]
                       [(y / 2) * vpx_img->stride[VPX_PLANE_U] + (x / 2)] = U;
        vpx_img->planes[VPX_PLANE_V]
                       [(y / 2) * vpx_img->stride[VPX_PLANE_V] + (x / 2)] = V;
      }
    }
  }
}

// Function to encode VPX image to data
int encode_frame(vpx_codec_ctx_t *codec, vpx_image_t *vpx_img, uint8_t **data,
                 size_t *data_size) {
  vpx_codec_err_t res;
  vpx_codec_iter_t iter = NULL;
  const vpx_codec_cx_pkt_t *pkt;

  res = vpx_codec_encode(codec, vpx_img, 0, 1, 0, VPX_DL_REALTIME);
  if (res) {
    return res;
  }

  while ((pkt = vpx_codec_get_cx_data(codec, &iter)) != NULL) {
    if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
      *data_size = pkt->data.frame.sz;
      *data = (uint8_t *)malloc(*data_size);
      if (*data == NULL) {
        return -1; // Memory allocation failed
      }
      memcpy(*data, pkt->data.frame.buf, *data_size);
      return 0;
    }
  }

  return -1;
}

// Function to convert ROS image to VPX image and encode it
int convert_and_encode(vpx_codec_ctx_t *codec,
                       const sensor_msgs__msg__Image *ros_img, uint8_t **data,
                       size_t *data_size) {
  vpx_image_t vpx_img;
  ros_to_vpx_image(ros_img, &vpx_img);

  int res = encode_frame(codec, &vpx_img, data, data_size);

  cleanup_vpx_image(&vpx_img);

  return res;
}

// Don't forget to clean up when done
void cleanup_vpx_image(vpx_image_t *vpx_img) {
  if (vpx_img->planes[VPX_PLANE_Y]) {
    free(vpx_img->planes[VPX_PLANE_Y]);
    vpx_img->planes[VPX_PLANE_Y] = NULL;
  }
  if (vpx_img->planes[VPX_PLANE_U]) {
    free(vpx_img->planes[VPX_PLANE_U]);
    vpx_img->planes[VPX_PLANE_U] = NULL;
  }
  if (vpx_img->planes[VPX_PLANE_V]) {
    free(vpx_img->planes[VPX_PLANE_V]);
    vpx_img->planes[VPX_PLANE_V] = NULL;
  }
}
