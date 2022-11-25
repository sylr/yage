// Copyright 2021 Google LLC
// Copyright 2021 Sylvain Rabot
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package main

import (
	"os"

	yagecmd "sylr.dev/yage/cmd"
)

func main() {
	if err := yagecmd.YAGECmd.Execute(); err != nil {
		os.Exit(1)
	}
}
