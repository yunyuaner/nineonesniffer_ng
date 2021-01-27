all: sniffer 

sniffer: sniffer.go ../nineonesniffer.go
	go build $<

clean:
	@rm -rf sniffer

.PHONY: clean all
