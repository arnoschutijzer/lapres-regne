These fixtures are eight-second synthetic sine waves generated with FFmpeg:

- `tone.mp3`: 440 Hz, 22050 Hz mono, MP3 at 48 kb/s.
- `tone.ogg`: 550 Hz, 48000 Hz stereo, OGG/Vorbis.
- `tone.flac`: 660 Hz, 32000 Hz mono, FLAC.

They exercise decoding, resampling, and channel conversion without a soundtrack
dependency. Tests do not require FFmpeg.
