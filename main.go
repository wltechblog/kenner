package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var directionVectors = map[string][2]float64{
	"north":     {0, -1},
	"south":     {0, 1},
	"east":      {1, 0},
	"west":      {-1, 0},
	"northeast": {0.7071, -0.7071},
	"northwest": {-0.7071, -0.7071},
	"southeast": {0.7071, 0.7071},
	"southwest": {-0.7071, 0.7071},
}

var validDirections = []string{"north", "south", "east", "west", "northeast", "northwest", "southeast", "southwest"}

func main() {
	input := flag.String("input", "", "Input image path (required)")
	output := flag.String("output", "", "Output video path (default: <input>_kenburns.mp4)")
	duration := flag.Float64("duration", 10, "Duration in seconds")
	fps := flag.Int("fps", 25, "Frames per second")
	focalX := flag.Int("focal-x", -1, "Focal point X (-1 = random)")
	focalY := flag.Int("focal-y", -1, "Focal point Y (-1 = random)")
	direction := flag.String("direction", "", "Pan direction (random if empty)")
	zoomEnd := flag.Float64("zoom", 1.3, "End zoom level (e.g. 1.3 = 30%% zoom in)")
	outWidth := flag.Int("width", 1920, "Output width")
	outHeight := flag.Int("height", 0, "Output height (0 = 16:9 from width)")
	scaleUp := flag.Int("scale-up", 0, "Internal scale-up factor (0 = auto)")

	flag.Parse()

	if *input == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *output == "" {
		ext := filepath.Ext(*input)
		base := strings.TrimSuffix(filepath.Base(*input), ext)
		dir := filepath.Dir(*input)
		*output = filepath.Join(dir, base+"_kenburns.mp4")
	}

	rand.Seed(time.Now().UnixNano())

	*direction = strings.ToLower(strings.TrimSpace(*direction))
	if *direction == "" {
		*direction = validDirections[rand.Intn(len(validDirections))]
	}
	dirVec, ok := directionVectors[*direction]
	if !ok {
		log.Fatalf("invalid direction %q; use one of: %s", *direction, strings.Join(validDirections, ", "))
	}

	imgW, imgH, err := getImageDimensions(*input)
	if err != nil {
		log.Fatalf("detecting image dimensions: %v", err)
	}
	fmt.Printf("Image: %dx%d\n", imgW, imgH)

	if *outHeight == 0 {
		*outHeight = int(math.Round(float64(*outWidth) * 9.0 / 16.0))
	}
	*outHeight += *outHeight % 2

	if *focalX < 0 || *focalX >= imgW {
		*focalX = rand.Intn(imgW)
	}
	if *focalY < 0 || *focalY >= imgH {
		*focalY = rand.Intn(imgH)
	}
	fmt.Printf("Focal point: (%d, %d)\n", *focalX, *focalY)
	fmt.Printf("Direction: %s, Zoom: %.2fx\n", *direction, *zoomEnd)

	totalFrames := int(float64(*fps) * *duration)

	cropX, cropY, cropW, cropH := computeCrop(imgW, imgH, *outWidth, *outHeight, *focalX, *focalY)

	bigW := *scaleUp
	if bigW <= 0 {
		bigW = *outWidth * int(math.Ceil(*zoomEnd+1))
		if bigW < 8000 {
			bigW = 8000
		}
	}

	focalScaledX := float64(*focalX-cropX) * float64(bigW) / float64(cropW)
	focalScaledY := float64(*focalY-cropY) * float64(bigW) / float64(cropW) * float64(cropH) / float64(cropH)

	maxPanX := (float64(bigW) - float64(bigW)/ *zoomEnd) * 0.4
	maxPanY := maxPanX * float64(*outHeight) / float64(*outWidth)

	startCX := focalScaledX - dirVec[0]*maxPanX
	startCY := focalScaledY - dirVec[1]*maxPanY

	cmd, filterComplex := buildFFmpegCmd(*input, *output, cropX, cropY, cropW, cropH, bigW,
		*outWidth, *outHeight, *fps, totalFrames, *zoomEnd,
		startCX, startCY, focalScaledX, focalScaledY, *duration)

	fmt.Printf("Running ffmpeg...\n")
	fmt.Printf("  Filter: %s\n", filterComplex)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("ffmpeg failed: %v", err)
	}

	fmt.Printf("Done: %s\n", *output)
}

func getImageDimensions(path string) (int, int, error) {
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		path,
	).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output: %s", strings.TrimSpace(string(out)))
	}

	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing width: %w", err)
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing height: %w", err)
	}
	return w, h, nil
}

func computeCrop(imgW, imgH, outW, outH, focalX, focalY int) (cropX, cropY, cropW, cropH int) {
	outAR := float64(outW) / float64(outH)
	imgAR := float64(imgW) / float64(imgH)

	if imgAR > outAR {
		cropH = imgH
		cropW = int(math.Round(float64(imgH) * outAR))
		cropW += cropW % 2
	} else {
		cropW = imgW
		cropH = int(math.Round(float64(imgW) / outAR))
		cropH += cropH % 2
	}

	cropX = focalX - cropW/2
	cropY = focalY - cropH/2

	if cropX < 0 {
		cropX = 0
	}
	if cropX+cropW > imgW {
		cropX = imgW - cropW
	}
	if cropY < 0 {
		cropY = 0
	}
	if cropY+cropH > imgH {
		cropY = imgH - cropH
	}

	return
}

func fmtExpr(f float64) string {
	return strconv.FormatFloat(f, 'f', 6, 64)
}

func buildFFmpegCmd(input, output string,
	cropX, cropY, cropW, cropH, bigW,
	outW, outH, fps, totalFrames int,
	zoomEnd, startCX, startCY, endCX, endCY, duration float64,
) (*exec.Cmd, string) {
	scaleH := bigW * cropH / cropW
	scaleH += scaleH % 2

	zoomExpr := fmt.Sprintf("1+(%s-1)*on/%d", fmtExpr(zoomEnd), totalFrames)

	cxExpr := fmt.Sprintf("%s+(%s-%s)*on/%d",
		fmtExpr(startCX), fmtExpr(endCX), fmtExpr(startCX), totalFrames)
	cyExpr := fmt.Sprintf("%s+(%s-%s)*on/%d",
		fmtExpr(startCY), fmtExpr(endCY), fmtExpr(startCY), totalFrames)

	xExpr := fmt.Sprintf("max(0,min(%s+%s/(2*zoom)-iw/(2*zoom),iw-iw/zoom))",
		cxExpr, fmtExpr(float64(bigW)))
	yExpr := fmt.Sprintf("max(0,min(%s+%s/(2*zoom)-ih/(2*zoom),ih-ih/zoom))",
		cyExpr, fmtExpr(float64(scaleH)))

	filterComplex := fmt.Sprintf(
		"[0]crop=%d:%d:%d:%d,scale=%d:%d,setsar=1:1,zoompan=z='%s':x='%s':y='%s':d=%d:s=%dx%d:fps=%d[out]",
		cropW, cropH, cropX, cropY,
		bigW, scaleH,
		zoomExpr, xExpr, yExpr,
		totalFrames, outW, outH, fps,
	)

	args := []string{
		"-loop", "1",
		"-i", input,
		"-y",
		"-filter_complex", filterComplex,
		"-map", "[out]",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-r", strconv.Itoa(fps),
		"-t", fmtExpr(duration),
		output,
	}

	return exec.Command("ffmpeg", args...), filterComplex
}
