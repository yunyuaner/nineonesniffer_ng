all: main get_video_descriptor video_download

main: main.go ../nineonesniffer.go
	go build $<

get_video_descriptor: get_video_descriptor.go ../nineonesniffer.go
	go build $<

video_download: video_download.go ../nineonesniffer.go
	go build $<

clean:
	@rm -rf main

.PHONY: clean all
