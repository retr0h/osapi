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

@test "invoke client network dns get subcommand" {
	run go run ${PROGRAM} client network dns get --interface-name eth0

	[ "$status" -eq 0 ]
}

@test "invoke client network dns update subcommand" {
	run go run ${PROGRAM} client network dns update \
		--servers "1.1.1.1,8.8.8.8" \
		--search-domains "foo.bar,baz.qux" \
		--interface-name eth0

	[ "$status" -eq 0 ]
}

@test "invoke client network ping subcommand" {
	run go run ${PROGRAM} client network ping \
		--address "127.0.0.1"

	[ "$status" -eq 0 ]
}
