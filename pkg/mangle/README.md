# mangle

Mangle is a simplified version of bytes.Buffer that reduces the size of the
API and aims at producing a buffer that only mutates and subslices a given
buffer.

It's main purpose is use in text changes that don't increase length, such as
changing escape codes to their proper ASCII or UTF-8 values.