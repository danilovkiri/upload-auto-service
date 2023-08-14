# Upload Automated Processing Service

This service can be deployed and used for automated CLI- and AMQP-based validation and processing of DNA Microarray
genotyping data in 23andMe and similar file formats (further referred to as "upload data"). This service cannot be used
for processing WGS-derived VCF files, FASTQ files or other types of data.

## Requirements

1. a host machine with 12 CPUs, 32 GB RAM min, 50 GB SSD storage min and Docker installed
2. a built Docker image for processing (private and cannot be exposed here).
3. a directory with files:
   1. `dbsnp-153-hgvs-atlasids.sorted.split.hg38.vcf.gz`
   2. `dbsnp-153-hgvs-atlasids.sorted.split.hg38.vcf.gz.tbi`
   3. `hg38.amb`
   4. `hg38.dict`
   5. `hg38.fa.fai` 
   6. `hg38.ann`
   7. `hg38.fa`
   8. `hg38.pac`
   9. `hg38.2bit`
   10. `hg38.bwt`
   11. `hg38.fa.alt`
   12. `hg38.sa`
4. a directory with subdirectories:
   1. `data` — all of the files from subsection 3 go here
   2. `intermediate` — empty
   3. `raw_data` — contains empty directories `atlas_raw_data`, `external_raw_data`, `binary`
   4. `source` — empty

## Configuration

All variables can be stored in `.env` or/and `.env.local` files which must be put in the parent project directory. Note
that `.env.local` overrides `.env`.

### S3

1. `S3_ACCESS_KEY_ID` — for processed data storage
2. `S3_SECRET_ACCESS_KEY` — for processed data storage
3. `S3_ENDPOINT` — universal
4. `S3_REGION` — universal
5. `S3_BUCKET` — universal
6. `S3_FOLDER_INTERNAL` — for processed data storage
7. `S3_FOLDER_EXTERNAL` — for processed data storage
8. `S3_FOLDER_BINARY` — for processed data storage
9. `S3_BUCKET_UPLOAD` — for data from upload client
10. `S3_FOLDER_UPLOAD` — for data from upload client
11. `S3_ACCESS_KEY_ID_UPLOAD` — for data from upload client
12. `S3_SECRET_ACCESS_KEY_UPLOAD` — for data from upload client

### Postgres DB

1. `DATABASE_DSN` — DSN for the DB where the service will store its data

### Logging

1. `LOG_LEVEL` — logging level as integer where `0` is debug

### Docker

1. `DOCKER_IMAGE_NAME` — name of the Docker image built as specified in subsection 2 of Requirements
2. `DOCKER_MOUNT_DIR` — absolute path of the directory from subsection 4 of Requirements
3. `DOCKER_EXEC` — Docker executable absolute path (can be derived from executing `which docker` in shell)

### HTTP Server
1. `SERVER_ADDRESS`
2. `IDLE_TIMEOUT`
3. `READ_TIMEOUT`
4. `WRITE_TIMEOUT`

### AMQP client
1. `AMQP_ADDR`
2. `AMQP_VALIDATION_EXCHANGE_INPUT_NAME`
3. `AMQP_VALIDATION_EXCHANGE_OUTPUT_NAME`
4. `AMQP_PROCESSING_EXCHANGE_INPUT_NAME`
5. `AMQP_PROCESSING_EXCHANGE_OUTPUT_NAME`
6. `AMQP_VALIDATION_QUEUE_NAME`
7. `AMQP_PROCESSING_QUEUE_NAME`
8. `AMQP_RRS_QUEUE_NAME`

## Usage

### First time use

Clone the project and navigate to its directory, run:
```shell
go build -o ./bin/console
```
then migrate the DB:
```shell
bin/console storage:migrate
```

### Manual processing

Validate the file (use option `--dry-run` to not save any data in DB):
```shell
bin/console file:validate --user-id <userid> --file-path <filepath>
```

Upon successful validation you can run processing:
```shell
bin/console file:process --user-id <userid> --barcode <barcode>
```

Processing will error immediately if prior validation has been unsuccessful OR has been executed in dry-run mode.
Processing itself can be run in dry-run mode invoking the `--dry-run` CLI flag with dry-run mode meaning that no data
will be uploaded to S3 after processing.

### Automated processing

Run the HTTP server for enabling HTTP queries for product code:
```shell
bin/console http:serve
```

Run the AMQP consumer:
```shell
bin/console messenger:consume
```

## CLI commands description

**file:validate** — runs validation for a local file

**file:process** — runs processing for a local file

**http:serve** — starts HTTP server

**messenger:consume** — starts AMQP listener

**messenger:create** — creates and publishes a message to queue

**storage:reset** — drops all tables in DB

**storage:migrate** — creates all tables in DB

**user:info** — retrieves all data for one user from DB

**user:all** — retrieves all data for all users from DB

**user:delete** — removes all data for one user from DB

**user:reset** — resets all data for one user in DB

## HTTP server API

Swagger documentation is available at `/api/v1/doc/index.html` after executing `http:serve` CLI command. Swagger
docs can be regenerated by running:

```shell
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g ./internal/api/v1/rest/handlers/handlers.go
```

API is available at `/api/v1`. Two endpoints are now available:
1. `/api/v1/status/{userID}` — get processing status
The response is a json
```json
{"current_status": "status"}
```
with status being a string and having values `new`, `running`, `done`, `error`, `NA` and code 200.

2. `/api/v1/product/{userID}` — get product code
The response is a json
```json
{"product_code": "code"}
```
with status being a string and code 200.

Error codes for both endpoints include 400, 500, 404, 417 depending on the nature of the underlying error.

## AMQP, queues and models

AMQP server must be 3.12.2 or later to support per-queue acknowledgement timeout changing.

### validation task message

Validation task must be sent to exchange as declared in `AMQP_VALIDATION_EXCHANGE_INPUT_NAME` env variable. The message
must follow the scheme below. Note that all values are strings. Note that `some_file.txt` MUST exists under the same
name in `S3_FOLDER_UPLOAD` inside `S3_BUCKET_UPLOAD`.

```json
{
   "user_id":  "100",
   "file_name":  "some_file.txt"
}
```

### processing task message

Processing task must be sent to exchange as declared in `AMQP_PROCESSING_EXCHANGE_INPUT_NAME` env variable. The message
must follow the scheme. Note that all values are strings. Note that `some_file.txt` MUST exists under the same name in
`S3_FOLDER_UPLOAD` inside `S3_BUCKET_UPLOAD`.

```json
{
   "user_id":  "100",
   "file_name":  "some_file.txt",
   "barcode": "0000-0000"
}
```

### response message

Response messages are sent via exchanges to `AMQP_RRS_QUEUE_NAME` queue and follow the scheme:

```json
{
   "user_id": "100",
   "file_name": "some_file.txt",
   "rsp_type": "validation",
   "is_ready": true
}
```
```json
{
   "user_id": "100",
   "file_name": "some_file.txt",
   "rsp_type": "processing",
   "is_ready": true
}
```