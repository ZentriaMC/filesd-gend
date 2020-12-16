#!/bin/sh
set -euo pipefail

curl -X PUT -d @test/put.json http://127.0.0.1:5555/api/v1/configure
curl -X PATCH -d @test/patch.json http://127.0.0.1:5555/api/v1/configure
curl -X DELETE -d @test/delete.json http://127.0.0.1:5555/api/v1/configure
