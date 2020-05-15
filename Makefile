PACKAGE = kreate.go
GEN_FILES = .workDir *-best-*.mp4 *-first-*.mp4

all: build

build:
	wget -O example.mp4 http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_60fps_normal.mp4

run:
	go run $(PACKAGE)

clean:
	rm -rf $(GEN_FILES)
