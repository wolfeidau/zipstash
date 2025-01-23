APPNAME := zipstash
STAGE ?= dev
BRANCH ?= main

DEPLOY_CMD := sam deploy

GIT_HASH := $(shell git rev-parse --short HEAD)

.PHONY: deploy
deploy: docker-login docker-build docker-push deploy-api

.PHONY: lint
lint:
	@echo "--- lint"
	$(shell go env GOPATH)/bin/gosec ./...
	$(shell go env GOPATH)/bin/staticcheck ./...

.PHONY: test
test:
	@echo "--- test"
	go test -v -cover ./...

.PHONY: fieldalignment
fieldalignment:
	@echo "--- field alignment"
	fieldalignment -fix .

.PHONY: docker-login
docker-login:
	$(eval DOCKER_HOSTNAME := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/repository_hostname' --query 'Parameter.Value' --output text))
	@aws ecr get-login-password | docker login --username AWS --password-stdin $(DOCKER_HOSTNAME)

.PHONY: docker-build
docker-build:
	@echo "--- docker build"
	@docker build --build-arg APP_VERSION=${GIT_HASH} -t $(APPNAME):$(BRANCH)_${GIT_HASH} -f Dockerfile.server .

.PHONY: docker-push
docker-push:
	$(eval DOCKER_REPO := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/repository_uri' --query 'Parameter.Value' --output text))
	@docker tag $(APPNAME):$(BRANCH)_${GIT_HASH} $(DOCKER_REPO):$(BRANCH)_${GIT_HASH}
	@docker push $(DOCKER_REPO):$(BRANCH)_${GIT_HASH}

.PHONY: deploy-repository
deploy-repository:
	@echo "--- deploy stack $(APPNAME)-$(STAGE)-$(BRANCH)-repository"
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/app/repository.cfn.yml \
		--capabilities CAPABILITY_IAM \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-repository \
		--parameter-overrides AppName=$(APPNAME) Stage=$(STAGE) Branch=$(BRANCH)

.PHONY: deploy-api
deploy-api:
	@echo "--- deploy stack $(APPNAME)-$(STAGE)-$(BRANCH)-api"
	$(eval DOCKER_REPO := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/repository_uri' --query 'Parameter.Value' --output text))
	@sam deploy \
		--image-repository=$(DOCKER_REPO) \
		--no-fail-on-empty-changeset \
		--template-file sam/app/api.cfn.yml \
		--capabilities CAPABILITY_IAM \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-api \
		--parameter-overrides AppName=$(APPNAME) Stage=$(STAGE) Branch=$(BRANCH) \
			ContainerImageUri=$(DOCKER_REPO):$(BRANCH)_${GIT_HASH} \
			OtelExporterEndpoint=$(OTEL_EXPORTER_OTLP_ENDPOINT) \
			OtelExporterHeaders="$(OTEL_EXPORTER_HEADERS)"

.PHONY: logs
logs:
	@sam logs --stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-api --tail

.PHONY: mkcert
mkcert:
	@mkdir -p .certs
	@mkcert -cert-file .certs/cert.pem -key-file .certs/key.pem localhost 127.0.0.1 ::1
