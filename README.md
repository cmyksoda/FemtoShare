# FemtoShare

FemtoShare is a fork of [PicoShare](https://github.com/mtlynch/picoshare) by Michael Lynch that converts uploaded videos so they play in every browser.

[![Docker Image Size](https://img.shields.io/docker/image-size/cmyksoda/femtoshare/latest)](https://hub.docker.com/r/cmyksoda/femtoshare/)
[![Docker Pulls](https://img.shields.io/docker/pulls/cmyksoda/femtoshare.svg?maxAge=604800)](https://hub.docker.com/r/cmyksoda/femtoshare/)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/m/cmyksoda/FemtoShare)](https://github.com/cmyksoda/FemtoShare/commits/main)
[![GitHub last commit](https://img.shields.io/github/last-commit/cmyksoda/FemtoShare)](https://github.com/cmyksoda/FemtoShare/commits/main)
[![Contributors](https://img.shields.io/github/contributors/cmyksoda/FemtoShare)](https://github.com/cmyksoda/FemtoShare/graphs/contributors)
[![License](http://img.shields.io/:license-agpl-blue.svg?style=flat-square)](LICENSE)

## Overview

FemtoShare is a minimalist, self-hosted web app that makes it easy to share files on the web without restrictions on size, file type, or media format.

[![FemtoShare demo](https://raw.githubusercontent.com/cmyksoda/FemtoShare/main/docs/readme-assets/demo.gif)](https://raw.githubusercontent.com/cmyksoda/FemtoShare/main/docs/readme-assets/demo-full.gif)

## FemtoShare vs PicoShare

FemtoShare works just like PicoShare, with a few changes:

|                   | PicoShare                                                                                                                | FemtoShare                                                                                                                                         |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Videos**        | Shared exactly as uploaded, so some phone videos (HEVC, HDR, or very tall screen recordings) don't play in every browser | Converted to an MP4 that plays in every browser, which also removes metadata like location. The upload page shows the link once the video is ready |
| **Disk space**    | Space from deleted files stays reserved until you shrink the database by hand                                            | Space from deleted files and converted videos goes back to your disk right away                                                                    |
| **After updates** | Browsers and CDNs can keep using old scripts for a few hours                                                             | Each update gets fresh script URLs, so pages never mix old and new code                                                                            |
| **Look**          | Green pS icon                                                                                                            | Soft magenta fS icon                                                                                                                               |

Everything else is the same, including settings and the database, so you can switch from PicoShare by changing the Docker image to `cmyksoda/femtoshare`.

## Why FemtoShare?

There are a million services for sharing files, but none of them are quite like FemtoShare. Here are FemtoShare's advantages:

- **Direct download links**: FemtoShare gives you a direct download link you can share with anyone. They can view or download the file with no ads or signups.
- **No file restrictions**: Unlike sites like imgur, Vimeo, or SoundCloud that only allow you to share specific types of files, FemtoShare lets you share any file of any size.
- **Videos that play everywhere**: FemtoShare converts uploaded videos to MP4 so they play in every browser, and the upload page holds back the link until the video is ready. Other files are never resized or re-encoded.

## Run FemtoShare

### From source

```bash
PS_SHARED_SECRET=somesecretpass PORT=4001 \
  go run cmd/femtoshare/main.go
```

Video conversion requires `ffmpeg` and `ffprobe` on your `PATH`. The Docker image includes them.

### From Docker

To run FemtoShare within a Docker container, mount a volume from your local system to store the FemtoShare sqlite database.

```bash
docker run \
  --env "PORT=4001" \
  --env "PS_SHARED_SECRET=somesecretpass" \
  --publish 4001:4001/tcp \
  --volume "${PWD}/data:/data" \
  --name femtoshare \
  cmyksoda/femtoshare
```

### From Docker + cloud data replication

If you specify settings for a [Litestream](https://litestream.io/)-compatible cloud storage location, FemtoShare will automatically replicate your data.

You can kill the container and start it later, and FemtoShare will restore your data from the cloud storage location and continue as if there was no interruption.

```bash
PORT=4001
PS_SHARED_SECRET="somesecretpass"
LITESTREAM_BUCKET=YOUR-LITESTREAM-BUCKET
LITESTREAM_ENDPOINT=YOUR-LITESTREAM-ENDPOINT
LITESTREAM_ACCESS_KEY_ID=YOUR-ACCESS-ID
LITESTREAM_SECRET_ACCESS_KEY=YOUR-SECRET-ACCESS-KEY

docker run \
  --publish "${PORT}:${PORT}/tcp" \
  --env "PORT=${PORT}" \
  --env "PS_SHARED_SECRET=${PS_SHARED_SECRET}" \
  --env "LITESTREAM_ACCESS_KEY_ID=${LITESTREAM_ACCESS_KEY_ID}" \
  --env "LITESTREAM_SECRET_ACCESS_KEY=${LITESTREAM_SECRET_ACCESS_KEY}" \
  --env "LITESTREAM_BUCKET=${LITESTREAM_BUCKET}" \
  --env "LITESTREAM_ENDPOINT=${LITESTREAM_ENDPOINT}" \
  --name femtoshare \
  cmyksoda/femtoshare
```

Notes:

- Only run one Docker container for each Litestream location.
  - FemtoShare can't sync writes across multiple instances.

### Using Docker Compose

To run FemtoShare under docker-compose, copy the following to a file called `docker-compose.yml` and then run `docker-compose up`.

```yaml
version: "3.2"
services:
  femtoshare:
    image: cmyksoda/femtoshare
    environment:
      - PORT=4001
      - PS_SHARED_SECRET=dummypass # Change to any password
    ports:
      - 4001:4001
    command: -db /data/store.db
    volumes:
      - ./data:/data
```

## Parameters

### Command-line flags

| Flag  | Meaning                 | Default Value     |
| ----- | ----------------------- | ----------------- |
| `-db` | Path to SQLite database | `"data/store.db"` |

### Environment variables

| Environment Variable    | Meaning                                                                                                            |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `PORT`                  | TCP port on which to listen for HTTP connections (defaults to 4001).                                               |
| `PS_BEHIND_PROXY`       | Set to `"true"` for better logging when FemtoShare is running behind a reverse proxy.                              |
| `PS_SHARED_SECRET`      | Specifies a passphrase for the admin user to log in to FemtoShare. Required if `PS_SHARED_SECRET_FILE` is not set. |
| `PS_SHARED_SECRET_FILE` | Path to a file containing the passphrase for the admin user. Required if `PS_SHARED_SECRET` is not set.            |

### Docker environment variables

You can adjust behavior of the Docker container by specifying these Docker-specific variables with `docker run -e`:

| Environment Variable           | Meaning                                                                                               |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- |
| `LITESTREAM_BUCKET`            | Litestream-compatible cloud storage bucket where Litestream should replicate data.                    |
| `LITESTREAM_ENDPOINT`          | Litestream-compatible cloud storage endpoint where Litestream should replicate data.                  |
| `LITESTREAM_ACCESS_KEY_ID`     | Litestream-compatible cloud storage access key ID to the bucket where you want to replicate data.     |
| `LITESTREAM_SECRET_ACCESS_KEY` | Litestream-compatible cloud storage secret access key to the bucket where you want to replicate data. |
| `LITESTREAM_PATH`              | Path within the Litestream bucket where Litestream should replicate data (defaults to `db`).          |
| `LITESTREAM_RETENTION`         | The amount of time Litestream snapshots & WAL files will be kept (defaults to 72h).                   |

### Docker build args

If you rebuild the Docker image from source, you can adjust the build behavior with `docker build --build-arg`:

| Build Arg            | Meaning                                                                     | Default Value |
| -------------------- | --------------------------------------------------------------------------- | ------------- |
| `litestream_version` | Version of [Litestream](https://litestream.io/) to use for data replication | `0.3.13`      |

## Deployment

FemtoShare is easy to deploy to cloud hosting platforms:

- [fly.io](docs/deployment/fly.io.md)
