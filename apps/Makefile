all: sniffer

sniffer: sniffer.go ../nineonesniffer.go ../configmgr.go
	go build $<

clean:
	@rm -rf sniffer

.PHONY: clean all
