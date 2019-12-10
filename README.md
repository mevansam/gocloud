# GoCloud

This golang library implements a set of abstractions for compute and storage for the most commonly available public cloud environments.

The API abstracts the following public clouds:

* Amazon Web Services
* Microsoft Azure
* Google Cloud Platform

The abstractions is implemented via the cloud interfaces in the `cloud` package. The cloud provider configurations closely follow the environment required by [Terraform](https://terraform.io). These abstractions are meant to complement the Terraform CLI and templates to provide cloud resource lifecycle management capabilities.