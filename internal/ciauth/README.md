# ciauth

This module is used to authenticate CI services which support OIDC. The service this powers has a trust relationship with the CI service, so the CI service can be trusted to provide the owner and repository, or project information.

# supported services

* GitHub Actions
* GitLab CI
* Buildkite

# usage

This module provides an `Interceptor` which can be used to authenticate a request with [connectrpc](https://connectrpc.com/).
