ROOT_DIR := .
EXPORTER_DIR := $(ROOT_DIR)/mongo-exporter
SENTIMENT_DIR := $(ROOT_DIR)/sentiment

EXPORT_FILE := test_data.txt
EXPORTED_DATA := $(ROOT_DIR)/$(EXPORT_FILE)
SENTIMENT_DATA := $(SENTIMENT_DIR)/$(EXPORT_FILE)

.PHONY: help export-data copy-data prepare-data run-heat run-crawler pipeline test

help:
	@echo "Targets:"
	@echo "  make export-data   - Export MongoDB data to ./test_data.txt"
	@echo "  make copy-data     - Copy ./test_data.txt to ./sentiment/test_data.txt"
	@echo "  make prepare-data  - Export + copy data"
	@echo "  make run-heat      - Start heat-service"
	@echo "  make run-crawler   - Start crawler-service"
	@echo "  make pipeline      - Prepare data then run crawler-service"
	@echo "  make test          - Run go test for sentiment modules"

export-data:
	cd $(EXPORTER_DIR) && go run main.go --config config.yaml --once

copy-data:
	@test -f $(EXPORTED_DATA) || (echo "missing $(EXPORTED_DATA), run 'make export-data' first" && exit 1)
	cp -f $(EXPORTED_DATA) $(SENTIMENT_DATA)
	@echo "copied to $(SENTIMENT_DATA)"

prepare-data: export-data copy-data

run-heat:
	cd $(SENTIMENT_DIR)/heat-service && go run cmd/main.go

run-crawler:
	cd $(SENTIMENT_DIR)/crawler-service && go run cmd/main.go

pipeline: prepare-data run-crawler

test:
	cd $(SENTIMENT_DIR)/pkg && go test ./...
	cd $(SENTIMENT_DIR)/crawler-service && go test ./...
	cd $(SENTIMENT_DIR)/heat-service && go test ./...
