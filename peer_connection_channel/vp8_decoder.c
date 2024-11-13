// vp8_decoder.c
#include "vp8_decoder.h"
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

// Function to copy the decoded frame to a GoCV Mat
void copy_frame_to_mat(vpx_image_t *img, unsigned char *dest,
                       unsigned int width, unsigned int height) {
  for (int y = 0; y < height; y++) {
    for (int x = 0; x < width; x++) {
      // Calculate correct indices using strides
      int y_idx = y * img->stride[VPX_PLANE_Y] + x;
      int u_idx = (y >> 1) * img->stride[VPX_PLANE_U] + (x >> 1);
      int v_idx = (y >> 1) * img->stride[VPX_PLANE_V] + (x >> 1);

      // Get YUV values
      int Y = img->planes[VPX_PLANE_Y][y_idx];
      int U = img->planes[VPX_PLANE_U][u_idx] - 128;
      int V = img->planes[VPX_PLANE_V][v_idx] - 128;

      // YUV to RGB conversion with proper coefficients
      int R = Y + (1.403 * V);
      int G = Y - (0.344 * U) - (0.714 * V);
      int B = Y + (1.770 * U);

      // Clamp values to [0, 255]
      R = R < 0 ? 0 : (R > 255 ? 255 : R);
      G = G < 0 ? 0 : (G > 255 ? 255 : G);
      B = B < 0 ? 0 : (B > 255 ? 255 : B);

      // Write to destination in BGR order
      int dest_idx = (y * width + x) * 3;
      dest[dest_idx + 0] = (unsigned char)B;
      dest[dest_idx + 1] = (unsigned char)G;
      dest[dest_idx + 2] = (unsigned char)R;
    }
  }
}
