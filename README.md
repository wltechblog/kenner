# kenner

A command-line tool for applying the Ken Burns pan/zoom effect to still images, producing video output via `ffmpeg`.

## Features

- Smooth zoom + pan animation on any still image
- Handles landscape, portrait, and wide panoramic images automatically
- Crops to output aspect ratio without letterboxing or empty space
- Configurable focal point, pan direction, zoom level, duration, and resolution
- Random focal point and direction when not specified

## Requirements

- [Go](https://golang.org/) 1.16+
- [ffmpeg](https://ffmpeg.org/) (with `ffprobe`)

## Build

```bash
go build -o kenner .
```

## Usage

```bash
./kenner -input <image> [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | *(required)* | Input image path |
| `-output` | `<input>_kenburns.mp4` | Output video path |
| `-duration` | `10` | Video length in seconds |
| `-fps` | `25` | Frames per second |
| `-width` | `1920` | Output width in pixels |
| `-height` | `0` (auto 16:9) | Output height in pixels |
| `-focal-x` | random | Focal point X coordinate |
| `-focal-y` | random | Focal point Y coordinate |
| `-direction` | random | Pan direction |
| `-zoom` | `1.3` | End zoom level (e.g. `1.3` = 30% zoom) |
| `-scale-up` | auto | Internal upscale factor |

### Directions

`north`, `south`, `east`, `west`, `northeast`, `northwest`, `southeast`, `southwest`

## Examples

### Basic usage (random focal point and direction)

```bash
./kenner -input photo.jpg
```

Produces `photo_kenburns.mp4` — a 10-second, 1920x1080 video with a random pan/zoom.

### Specify direction and duration

```bash
./kenner -input landscape.jpg -duration 15 -direction southeast
```

15-second video panning southeast with a gradual 1.3x zoom.

### Portrait image, panning north

```bash
./kenner -input portrait.jpg -direction north -zoom 1.5
```

Automatically crops to 16:9, zooms to 1.5x while panning upward.

### Custom focal point and output size

```bash
./kenner -input photo.jpg -focal-x 960 -focal-y 540 -width 1280 -height 720 -duration 8
```

Zooms into the center of a 1920x1080 image, outputting 1280x720 at 8 seconds.

### Wide panoramic banner

```bash
./kenner -input banner.png -direction east -width 1280 -zoom 1.4
```

Crops the wide image to 16:9 and pans eastward.

## How It Works

1. Detects input image dimensions via `ffprobe`
2. Crops to the target output aspect ratio, centering on the focal point
3. Scales up the cropped image to provide room for zooming
4. Constructs an `ffmpeg` zoompan filter with computed expressions for smooth linear interpolation of zoom level and pan position
5. Encodes to H.264 MP4

## License

[GPL-2.0](LICENSE)
