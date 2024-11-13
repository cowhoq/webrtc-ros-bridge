// vp8_decoder.c
#include <vpx/vp8.h>
#include <vpx/vp8dx.h>
#include <vpx/vpx_decoder.h>
#include "vp8_decoder.h"

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
  return vpx_codec_get_frame(codec, NULL);
}
