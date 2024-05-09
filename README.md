# Elasticsearch Index Pruner

## Overview

The Elasticsearch Index Pruner is a tool designed to manage the size of indices in an Elasticsearch cluster.

It is made to address the specific issue of ILM not supporting rotation by size. This tool allows you to specify a maximum size for each index alias, and it will automatically prune indices under that alias to ensure they do not exceed the specified size limit.

## Features

- **Alias Management**: Automatically manages indices under specified aliases to ensure they do not exceed predefined size limits.
- **Flexible Configuration**: Supports configuration via a YAML file and environment variables.
- **Secure Connection**: Optionally configure SSL/TLS connections including CA certificates and skip verification for development environments.

## Configuration

### YAML Configuration File

The application is configured via a YAML file that specifies details about the Elasticsearch cluster and the index aliases to manage. Below is an example of the configuration structure:

```yaml
cluster:
  url: "https://elasticsearch.example.com:9200"
  username: "elastic"
  password: "password"
  ca_cert_path: "/path/to/ca.pem"
  skip_tls_verify: false

aliases:
  - name: "logs"
    max_size: "10GB"
  - name: "metrics"
    max_size: "20GB"
```

#### Configuration Details:

- `url`: The URL of the Elasticsearch cluster.
- `username`: Username for basic auth.
- `password`: Password for basic auth.
- `ca_cert_path`: Path to the CA certificate for SSL verification.
- `skip_tls_verify`: Set to `true` to skip TLS verification (useful for development environments).
- `aliases`: A list of aliases to manage with their respective size limits.

### Environment Variables

The application can also be configured using environment variables. These will override settings provided in the YAML file.

| Environment Variable     | Description                              |
|--------------------------|------------------------------------------|
| `ES_CLUSTER_URL`         | URL of the Elasticsearch cluster.        |
| `ES_USERNAME`            | Username for basic auth.                 |
| `ES_PASSWORD`            | Password for basic auth.                 |
| `ES_CA_CERT_PATH`        | Path to the CA certificate.              |
| `ES_SKIP_TLS_VERIFY`     | Skip TLS verification (`true` or `false`). |

### Running the Application

To run the application, first ensure that the configuration file is set up correctly. You can then start the application with the following command:

```bash
go run main.go
```

Make sure that the main executable points to the configuration loader and the monitoring setup as documented in the codebase.

### Building and Deployment

To build the application, use the standard Go build command:

```bash
go build -o es-index-pruner
```

Run the compiled binary with:

```bash
./es-index-pruner
```