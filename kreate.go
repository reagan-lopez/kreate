package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DEFAULT_TRIM_START      = "0"
	DEFAULT_TRIM_DURATION   = "60"
	DEFAULT_PART_DURATION   = "10"
	DEFAULT_INPUT_FILENAME  = "example.mp4"
	DEFAULT_TRIM_DIR_PREFIX = ".output"
	DEFAULT_VIDEO_LIST_FILE = ".videolist.txt"
)

/*
wget -O example.mp4 http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_60fps_normal.mp4
ffmpeg -fflags +genpts -i example.mp4 -map 0 -c copy -f segment -segment_format mp4 -segment_time 30 -segment_list video.ffcat -reset_timestamps 1 -v error chunk-%03d.mp4
ffmpeg -y -v error -i video.ffcat -map 0 -c copy output.mp4
*/

func main() {
	inputName := flag.String("i", DEFAULT_INPUT_FILENAME, "input file name")
	trimStart := flag.String("s", DEFAULT_TRIM_START, "trim start time in seconds")
	trimDuration := flag.String("d", DEFAULT_TRIM_DURATION, "trim duration in seconds")
	partDuration := flag.String("p", DEFAULT_PART_DURATION, "part duration in seconds")
	parallel := flag.Bool("pp", false, "execute in parallel")
	flag.Parse()
	log.Println("Flag values: ", *inputName, *trimStart, *trimDuration, *partDuration, *parallel)

	TrimVideoFirst(*inputName, *trimDuration)

	workDir := DEFAULT_TRIM_DIR_PREFIX + "-" + strings.TrimSuffix(*inputName, filepath.Ext(*inputName)) + "-" + time.Now().Format("20060102150405")
	TrimVideoBest(*inputName, *trimStart, *trimDuration, *partDuration, workDir)

	os.RemoveAll(workDir)
}

func TrimVideoFirst(inputName, trimDuration string) {
	outputName := strings.Replace(inputName, filepath.Ext(inputName), "-"+"first"+"-"+time.Now().Format("20060102150405")+filepath.Ext(inputName), 1)
	trimVideo(inputName, "0", trimDuration, outputName)
}

func TrimVideoBest(inputName, trimStart, trimDuration, partDuration, workDir string) {
	var err error
	mkdirHard(workDir)

	td, err := strconv.Atoi(trimDuration)
	check(err)
	pd, err := strconv.Atoi(partDuration)
	check(err)
	parts := td / pd

	inputDuration := getVideoDuration(inputName)
	id, err := strconv.ParseFloat(inputDuration, 64)
	check(err)
	gd := int(id) / parts

	ts, err := strconv.Atoi(trimStart)
	check(err)
	for p := 1; p <= parts; p++ {
		start := strconv.Itoa(ts)
		outputName := strconv.Itoa(p) + filepath.Ext(inputName)
		outputPath := filepath.Join(workDir, outputName)
		trimVideo(inputName, start, partDuration, outputPath)
		ts += gd + pd
	}

	createVideoListFile(workDir)
	outputName := strings.Replace(inputName, filepath.Ext(inputName), "-"+"best"+"-"+time.Now().Format("20060102150405")+filepath.Ext(inputName), 1)
	JoinVideo(workDir, outputName)
}

func JoinVideo(workDir, outputName string) {
	var err error
	cmd := exec.Command(
		"ffmpeg",
		"-f", "concat", "-safe", "0", "-i", filepath.Join(workDir, DEFAULT_VIDEO_LIST_FILE),
		"-c", "copy", outputName,
	)
	log.Println("Join command: ", cmd)
	_, err = cmd.Output()
	check(err)
}

// Returns duration in seconds. E.g. 2018.368000
func getVideoDuration(inputName string) string {
	var err error

	cmd := exec.Command(
		"ffprobe",
		"-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1",
		inputName,
	)
	log.Println("Duration command: ", cmd)

	out, err := cmd.Output()
	check(err)

	duration := strings.TrimSuffix(string(out), "\n")

	regexAlpha := regexp.MustCompile("[h,m]")
	// Format Trim Duration upto millisecond level. E.g. 01:33:38.368
	td, err := strconv.ParseFloat(duration, 64)
	check(err)
	d, err := time.ParseDuration((time.Duration(td*1000) * time.Millisecond).String())
	check(err)
	totalDuration := regexAlpha.ReplaceAllString(d.String(), ":")
	totalDuration = strings.TrimSuffix(totalDuration, "s")
	log.Println("Total Duration: " + totalDuration)

	return duration
}

func trimVideo(inputName, trimStart, trimDuration, outputName string) {
	var err error
	regexAlpha := regexp.MustCompile("[h,m]")

	// Format Trim Start upto millisecond level. E.g. 00:00:00.000
	ts, err := strconv.ParseFloat(trimStart, 64)
	check(err)
	s, err := time.ParseDuration((time.Duration(ts*1000) * time.Millisecond).String())
	check(err)
	start := regexAlpha.ReplaceAllString(s.String(), ":")
	start = strings.TrimSuffix(start, "s")

	// Format Trim Duration upto millisecond level. E.g. 01:33:38.368
	td, err := strconv.ParseFloat(trimDuration, 64)
	check(err)
	d, err := time.ParseDuration((time.Duration(td*1000) * time.Millisecond).String())
	check(err)
	duration := regexAlpha.ReplaceAllString(d.String(), ":")
	duration = strings.TrimSuffix(duration, "s")

	log.Println("Trim Start: " + start)
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputName,
		"-ss", start,
		"-t", duration,
		"-c", "copy",
		outputName,
	)
	log.Println("Trim command: ", cmd)

	_, err = cmd.Output()
	check(err)
}

func createVideoListFile(workDir string) {
	var err error
	f, err := os.Create(filepath.Join(workDir, DEFAULT_VIDEO_LIST_FILE))
	check(err)
	defer f.Close()

	_, err = os.Stat(workDir)
	check(err)

	fileList := make(map[int]string)

	err = filepath.Walk(workDir, func(path string, fi os.FileInfo, err error) error {
		if filepath.Ext(path) == ".mp4" {
			key, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
			check(err)

			fullPath, err := filepath.Abs(path)
			check(err)

			fileList[key] = "file " + fullPath + "\n"

			check(err)
		}
		return nil
	})
	check(err)

	keys := make([]int, 0, len(fileList))
	for k := range fileList {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		_, err = f.Write([]byte(fileList[k]))
		check(err)
	}
}

func mkdirHard(path string) {
	var err error

	err = os.RemoveAll(path)
	check(err)

	err = os.Mkdir(path, os.ModePerm)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
