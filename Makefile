all: depends release


depends:
	@/usr/bin/tsc --version >/dev/null || (echo "missing typescript compiler"; exit 1)
	@/usr/bin/go version >/dev/null || (echo "missing golang"; exit 1)

release: depends
	/usr/bin/tsc
	/usr/bin/go build -v cmd

debug: depends
	/usr/bin/tsc
	/usr/bin/go run ./cmd
