all: depends release

depends:
	@$$(which tsc) --version >/dev/null || (echo "missing typescript compiler"; exit 1)
	@$$(which go) version >/dev/null || (echo "missing golang"; exit 1)
	@$$(which curl) --version >/dev/null || (echo "missing curl"; exit 1)
	@$$(which 7z) >/dev/null || (echo "missing p7zip"; exit 1)
ifeq ("$(wildcard './site/css/external/fontawesome6')", "")
	@mkdir -p ./site/css/external/
	@curl -L https://use.fontawesome.com/releases/v6.2.1/fontawesome-free-6.2.1-web.zip -o site/css/external/fontawesome6.zip
	@7z x site/css/external/fontawesome6.zip -o"site/css/external/fontawesome6"
	@mv site/css/external/fontawesome6/fontawesome-free-6.2.1-web/* site/css/external/fontawesome6/
	@rmdir site/css/external/fontawesome6/fontawesome-free-6.2.1-web/
	@rm -f site/css/external/fontawesome6.zip
endif
	@go get -v ./...

release: depends
	@$$(which tsc)
	@$$(which go) build -o "build/$(MEALBOT_PLATFORM)/mealbot" -v ./cmd

debug: depends
	@$$(which tsc)
	@$$(which go) run ./cmd

container: release
	@$$(which docker) build . -t bowerscd/special-tribble:latest

pkg:


clean:
	@rm -rf build
