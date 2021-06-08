all: sniffer

SOURCE = ../nineonesniffer.go ../configmgr.go ../persister.go ../parser.go ../fetcher.go ../dataset.go ../algorithm.go ../obscurer.go

sniffer: sniffer.go $(SOURCE)
	go build $<

clean:
	@rm -rf sniffer

.PHONY: clean all test
