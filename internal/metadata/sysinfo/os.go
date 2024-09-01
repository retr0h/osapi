// Copyright (c) 2024 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package sysinfo

import (
	"strings"

	"gopkg.in/ini.v1"
)

// TODO(retr0h): should it act like the unix package?
// 	"golang.org/x/sys/unix"
//	var loadAvg unix.Loadavg
//	err := unix.Getloadavg(&loadAvg)

// GetOSInfo get Operating System Information.
func (si *SysInfo) GetOSInfo() *OS {
	const osReleaseFile = "/etc/os-release"

	file, err := si.appFs.Open(osReleaseFile)
	if err != nil {
		return &OS{}
	}
	defer func() { _ = file.Close() }()

	iniFile, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, file)
	if err != nil {
		return &OS{}
	}

	distribution := iniFile.Section("").Key("NAME").String()
	version := iniFile.Section("").Key("VERSION_ID").String()

	return &OS{
		Distribution: strings.ToLower(distribution),
		Version:      strings.ToLower(version),
	}
}
