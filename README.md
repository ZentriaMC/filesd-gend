# filesd-gend

Generates a JSON file for file-based service discovery functionality for [Prometheus][file-sd].

## Usage

Start up `filesd-gend -sd-file /tmp/sd.json`

Within Prometheus configuration, define:

```yaml
scrape_configs:
  - job_name: "node"
    file_sd_configs:
    - files:
      - "/tmp/sd.json"
```

Add scrape target using PUT HTTP request (listening on 127.0.0.1)

```sh
$ curl -X PUT -d '@-' http://127.0.0.1:5555/api/v1/configure <<EOF
{
	"target_id": "00000000-0000-0000-0000-000000000000",
	"targets": [
		"127.0.0.1:2112"
	],
	"labels": {
		"label_1": "1234"
	}
}
EOF
```

## License

GNU General Public License version 3

<!-- links -->
[file-sd]: https://prometheus.io/docs/guides/file-sd/
