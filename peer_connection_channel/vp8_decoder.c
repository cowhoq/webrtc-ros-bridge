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
  // Assuming img->planes[0] contains the Y plane data (for YUV format)
  // and we are converting it to a BGR format for GoCV

  int y_index = 0;
  int uv_index = 0;
  for (int j = 0; j < height; j++) {
    for (int i = 0; i < width; i++) {
      // Y value
      unsigned char Y = img->planes[VPX_PLANE_Y][y_index];
      // U and V values (assuming they are subsampled)
      unsigned char U = img->planes[VPX_PLANE_U][uv_index];
      unsigned char V = img->planes[VPX_PLANE_V][uv_index];

      // Convert YUV to BGR
      unsigned char B = Y + 1.772 * (U - 128);
      unsigned char G = Y - 0.344136 * (U - 128) - 0.714136 * (V - 128);
      unsigned char R = Y + 1.402 * (V - 128);

      // Store in the destination array (BGR format)
      dest[(j * width + i) * 3 + 0] = B; // B
      dest[(j * width + i) * 3 + 1] = G; // G
      dest[(j * width + i) * 3 + 2] = R; // R

      y_index++;
      if (i % 2 == 0 && j % 2 == 0) {
        uv_index++; // Increment UV index for subsampling
      }
    }
  }
}
