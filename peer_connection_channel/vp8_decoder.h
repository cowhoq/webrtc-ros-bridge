#ifndef VPX_DECODER_H_
#define VPX_DECODER_H_

#include <vpx/vp8.h>
#include <vpx/vp8dx.h>
#include <vpx/vpx_decoder.h>

int init_decoder(vpx_codec_ctx_t *codec, unsigned int w, unsigned int h);
int decode_frame(vpx_codec_ctx_t *codec, const uint8_t *data, size_t data_size);
vpx_image_t *get_frame(vpx_codec_ctx_t *codec);

#endif // VPX_DECODER_H_
