.PHONY: all build install gen-test test clean

all: build

build:
	go build -o bin/inpupa ./cmd/inpupa

install:
	go install ./cmd/inpupa

# Generate .inpupa.gen.go files for the test tree (invalid directories are excluded: they contain intentionally broken annotations).
gen-test: build
	./bin/inpupa --input test --exclude invalid

test: gen-test
	go test ./...

clean:
	rm -f bin/inpupa
	find test -name '*.inpupa.gen.go' -delete
