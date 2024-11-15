// vp8_decoder.c
#include "vp8_decoder.h"
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

// #define 永远缅怀
#ifdef 永远缅怀
void copy_frame_to_mat(vpx_image_t *img, unsigned char *dest,
                       unsigned int width, unsigned int height) {
  for (int y = 0; y < height; y++) {
    for (int x = 0; x < width; x += 8) { // Process 8 pixels at a time
      // Load Y values
      __m128i Y = _mm_loadl_epi64(
          (__m128i *)&img
              ->planes[VPX_PLANE_Y][y * img->stride[VPX_PLANE_Y] + x]);

      // Load U and V values (subsampled by 2)
      __m128i U = _mm_loadl_epi64(
          (__m128i *)&img
              ->planes[VPX_PLANE_U]
                      [(y >> 1) * img->stride[VPX_PLANE_U] + (x >> 1)]);
      __m128i V = _mm_loadl_epi64(
          (__m128i *)&img
              ->planes[VPX_PLANE_V]
                      [(y >> 1) * img->stride[VPX_PLANE_V] + (x >> 1)]);

      // Unpack U and V values to 16-bit integers and subtract 128
      U = _mm_sub_epi16(_mm_unpacklo_epi8(U, _mm_setzero_si128()),
                        _mm_set1_epi16(128));
      V = _mm_sub_epi16(_mm_unpacklo_epi8(V, _mm_setzero_si128()),
                        _mm_set1_epi16(128));

      // Unpack Y values to 16-bit integers
      Y = _mm_unpacklo_epi8(Y, _mm_setzero_si128());

      // YUV to RGB conversion
      __m128i R = _mm_add_epi16(
          Y, _mm_mulhi_epi16(V, _mm_set1_epi16(1436))); // 1.403 * 1024 = 1436
      __m128i G = _mm_sub_epi16(
          Y, _mm_add_epi16(
                 _mm_mulhi_epi16(U, _mm_set1_epi16(352)),
                 _mm_mulhi_epi16(
                     V, _mm_set1_epi16(
                            731)))); // 0.344 * 1024 = 352, 0.714 * 1024 = 731
      __m128i B = _mm_add_epi16(
          Y, _mm_mulhi_epi16(U, _mm_set1_epi16(1814))); // 1.770 * 1024 = 1814

      // Pack and clamp RGB values to 8-bit
      __m128i zero = _mm_setzero_si128();
      R = _mm_packus_epi16(R, zero);
      G = _mm_packus_epi16(G, zero);
      B = _mm_packus_epi16(B, zero);

      // Interleave RGB values
      __m128i RG = _mm_unpacklo_epi8(R, G);
      __m128i BZ = _mm_unpacklo_epi8(B, zero);
      __m128i RGB0 = _mm_unpacklo_epi16(RG, BZ);
      __m128i RGB1 = _mm_unpackhi_epi16(RG, BZ);

      // Store RGB values to destination
      _mm_storeu_si128((__m128i *)&dest[(y * width + x) * 3], RGB0);
      _mm_storeu_si128((__m128i *)&dest[(y * width + x + 4) * 3], RGB1);
    }
  }
}
#else
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
#endif
