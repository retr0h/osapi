# Copyright (c) 2024 John Dewey

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to
# deal in the Software without restriction, including without limitation the
# rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
# sell copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
# DEALINGS IN THE SOFTWARE.

load "libs/common.bash"

setup() {
	start_server
}

teardown() {
	stop_server
}

# NOTE(retr0h): subsequent tests rely on this one -- will refactor
@test "invoke client queue add subcommand" {
	run go run ${PROGRAM} client queue add -p proto/dns.bin

	[ "$status" -eq 0 ]
}

@test "invoke client queue delete subcommand" {
	run go run ${PROGRAM} client queue list --json

	id=$(echo "${output}" | jq -r '.response | fromjson | .items[0].id')

	run go run ${PROGRAM} client queue delete -m ${id}

	[ "$status" -eq 0 ]
}

@test "invoke client queue get subcommand" {
	run go run ${PROGRAM} client queue list --json

	id=$(echo "${output}" | jq -r '.response | fromjson | .items[0].id')

	run go run ${PROGRAM} client queue get -m ${id}

	[ "$status" -eq 0 ]
}

@test "invoke client queue list subcommand" {
	run go run ${PROGRAM} client queue list --json

	echo "${output}" | jq -e '.response | fromjson | .items | length > 0'

	[ "$status" -eq 0 ]
}
