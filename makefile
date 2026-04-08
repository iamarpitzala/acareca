dev:
	go run ./cmd/api

build:
	go build -o acareca ./cmd/api

swagger:
	swag init -g cmd/api/main.go

postman: docs/swagger.json
	@openapi2postmanv2 -s docs/swagger.json -o postman/collection.json

docs/swagger.json: swagger
	@:

vet:
	go vet ./...

seed:
	go run scripts/seed_clinics_forms.go

seed-advanced:
	go run scripts/seed_advanced.go

seed-large:
	go run scripts/seed_advanced.go -clinics 50 -forms 10 -verbose

seed-cleanup:
	go run scripts/cleanup_seed.go -confirm

seed-cleanup-dry:
	go run scripts/cleanup_seed.go -dry-run