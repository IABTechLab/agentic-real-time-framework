# Root-level repository tasks
# Keep the root Makefile focused on shared proto generation.

.PHONY: generate fetch-openrtb

LANG ?= go
LANGS ?= $(LANG)

generate: fetch-openrtb
	mkdir -p pkg
	./scripts/generate.sh --lang $(LANGS)

fetch-openrtb:
	mkdir -p proto/com/iabtechlab/openrtb/v2
	curl -L --fail \
		https://raw.githubusercontent.com/InteractiveAdvertisingBureau/openrtb2.x/main/proto/src/main/com/iabtechlab/openrtb/v2/openrtb.proto \
		-o proto/com/iabtechlab/openrtb/v2/openrtb.proto
