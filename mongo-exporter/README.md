# mongo-exporter

A lightweight Go subsystem that reads all documents from a MongoDB collection
and writes them as **JSONL** (one JSON object per line) to a configured `.txt`
file path on a repeating schedule.

Other subsystems simply read the output file — they never talk to MongoDB
directly.

---

## Project structure

```
mongo-exporter/
├── main.go                      # Entry point, CLI flags, graceful shutdown
├── config.yaml                  # All runtime configuration (edit this)
├── go.mod
├── Dockerfile
└── internal/
    ├── config/
    │   └── config.go            # Config loader & validation
    └── exporter/
        └── exporter.go          # MongoDB reader + JSONL writer + scheduler
```

---

## Configuration (`config.yaml`)

| Key | Default | Description |
|-----|---------|-------------|
| `mongodb.uri` | *(required)* | MongoDB connection URI |
| `mongodb.database` | *(required)* | Database name |
| `mongodb.collection` | *(required)* | Collection to export |
| `mongodb.timeout_seconds` | `30` | Timeout per MongoDB operation |
| `output.file_path` | *(required)* | Where to write the JSONL `.txt` file |
| `output.temp_suffix` | `.tmp` | Suffix for in-progress write (atomic) |
| `scheduler.interval` | `5m` | How often to run export (`30s`, `5m`, `1h`, …) |

---

## Running

### Install dependencies

```bash
go mod tidy
```

### Run continuously (scheduled)

```bash
go run main.go --config config.yaml
```

### Run once and exit

```bash
go run main.go --config config.yaml --once
```

### Build binary

```bash
go build -o mongo-exporter ./main.go
./mongo-exporter --config config.yaml
```

### Docker

```bash
docker build -t mongo-exporter .

# Mount the output directory so other subsystems can access the file
docker run \
  -v /host/path/to/output:/var/data/export \
  -v $(pwd)/config.yaml:/app/config.yaml \
  mongo-exporter
```

---

## Output format (JSONL)

Each line in `output.txt` is one complete JSON document from MongoDB:

```
{"_id":"64a1f...","name":"Alice","age":30,"active":true}
{"_id":"64a1f...","name":"Bob","age":25,"active":false}
```

- The output file is written **atomically**: a `.tmp` file is written first,
  then renamed. Reading subsystems will never see a partial file.
- The output directory is **created automatically** if it doesn't exist.

---

## How other subsystems read the file

Since the output is plain JSONL, any language can consume it line by line:

```python
# Python example
import json
with open("/var/data/export/output.txt") as f:
    for line in f:
        doc = json.loads(line)
        print(doc)
```

```go
// Go example
f, _ := os.Open("/var/data/export/output.txt")
scanner := bufio.NewScanner(f)
for scanner.Scan() {
    var doc map[string]any
    json.Unmarshal(scanner.Bytes(), &doc)
}
```

---

## MongoDB URI examples

| Scenario | URI |
|----------|-----|
| Local, no auth | `mongodb://localhost:27017` |
| Local, with auth | `mongodb://user:pass@localhost:27017/?authSource=admin` |
| Replica set | `mongodb://host1:27017,host2:27017/?replicaSet=rs0` |
| MongoDB Atlas | `mongodb+srv://user:pass@cluster.mongodb.net` |
