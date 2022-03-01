all: depends release


depends:
	@/usr/bin/tsc --version >/dev/null || (echo "missing typescript compiler"; exit 1)
	@/usr/bin/go version >/dev/null || (echo "missing golang"; exit 1)

release: depends
	/usr/bin/tsc
	mkdir -p build/
	find ./site -regextype egrep -iregex "^(.*/external/.*|.*\.(html|css|ts|js(\.map)))\$$" -exec install -D -m 600 "{}" "build/{}" \;
	/usr/bin/go build -o "build/mealbot" -v ./cmd

debug: depends
	/usr/bin/tsc
	/usr/bin/go run ./cmd

container: release
	docker build . -t bowerscd/special-tribble:latest

clean:
	rm -r build