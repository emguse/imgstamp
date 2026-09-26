# Third-Party Notices

Original imgstamp code, documentation, and project-generated fixtures are
licensed under the [MIT License](LICENSE). Third-party software and artwork
retain their original licenses; the root MIT license does not replace them.

Include both `LICENSE` and this complete `THIRD_PARTY_NOTICES.md` with binary
releases. Source distributions must also retain the fixture attribution and
license in `internal/pages/testdata/`. Update this inventory when dependencies,
the Go toolchain, or third-party assets change.

## Go modules

The versions below match `go.mod`. The application dependency lists for
macOS ARM64 and Windows x64 were inspected; `golang.org/x/sys` is used in the
Windows build. Test-only tooling dependencies of upstream modules are not
presented as application dependencies.

| Module | Version | License | Use |
| --- | --- | --- | --- |
| [github.com/hhrutter/tiff](https://pkg.go.dev/github.com/hhrutter/tiff@v1.0.6) | v1.0.6 | BSD-3-Clause | TIFF decoding |
| [github.com/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2@v2.4.3) | v2.4.3 | MIT | TOML configuration |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image@v0.46.0) | v0.46.0 | BSD-3-Clause | Image and font processing; Go Regular font in tests |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text@v0.42.0) | v0.42.0 | BSD-3-Clause | Indirect font encoding support |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys@v0.48.0) | v0.48.0 | BSD-3-Clause | Indirect Windows system support |

The Go runtime and standard library used to build the application are also
covered by the Go BSD-3-Clause license reproduced below (inspected with Go
1.27.1). The Go Regular test font is provided by `golang.org/x/image/font/gofont/goregular`;
it is used in tests and is not embedded in the application executable.

### github.com/pelletier/go-toml/v2 v2.4.3

The following notice is copied verbatim from the module's `LICENSE`:

```text
The MIT License (MIT)

go-toml v2
Copyright (c) 2021 - 2023 Thomas Pelletier

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### github.com/hhrutter/tiff v1.0.6

The following notice is copied verbatim from the module's `LICENSE` and is
also retained at `internal/pages/testdata/LICENSE` for the copied fixtures:

```text
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### Go runtime, standard library, and golang.org/x modules

The following identical notice is copied verbatim from Go 1.27.1 and the
`LICENSE` files of `golang.org/x/image v0.46.0`, `golang.org/x/text v0.42.0`,
and `golang.org/x/sys v0.48.0`:

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### Additional Go patent grant

The identical `PATENTS` notice shipped with Go and those three golang.org/x
modules is retained below:

```text
Additional IP Rights Grant (Patents)

"This implementation" means the copyrightable works distributed by
Google as part of the Go project.

Google hereby grants to You a perpetual, worldwide, non-exclusive,
no-charge, royalty-free, irrevocable (except as stated in this section)
patent license to make, have made, use, offer to sell, sell, import,
transfer and otherwise run, modify and propagate the contents of this
implementation of Go, where such license applies only to those patent
claims, both currently owned or controlled by Google and acquired in
the future, licensable by Google that are necessarily infringed by this
implementation of Go.  This grant does not include claims that would be
infringed only as a consequence of further modification of this
implementation.  If you or your agent or exclusive licensee institute or
order or agree to the institution of patent litigation against any
entity (including a cross-claim or counterclaim in a lawsuit) alleging
that this implementation of Go or any code incorporated within this
implementation of Go constitutes direct or contributory patent
infringement, or inducement of patent infringement, then any patent
rights granted to you under this License for this implementation of Go
shall terminate as of the date such litigation is filed.
```

## Go Regular font used in tests

Tests load the Go Regular font from `golang.org/x/image/font/gofont/goregular`.
The font has its own Bigelow & Holmes copyright notice. The following license
text is reproduced from the font's embedded OpenType name-table license record
(name ID 13). It is not a bundled application font.

```text
Copyright (c) 2016 Bigelow & Holmes Inc.. All rights reserved.

Distribution of this font is governed by the following license. If you do not agree to this license, including the disclaimer, do not distribute or modify this font.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

   * Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

   * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

   * Neither the name of Google Inc. nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

DISCLAIMER: THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Go gopher artwork in test fixtures

The Go gopher was designed by **Renee French**. The original design is licensed
under [Creative Commons Attribution 4.0 International (CC BY 4.0)](https://creativecommons.org/licenses/by/4.0/)
([legal code](https://creativecommons.org/licenses/by/4.0/legalcode)).
See the [official Go gopher attribution](https://go.dev/doc/gopher/README)
and [The Go Gopher](https://go.dev/blog/gopher).

The following monochrome fixture versions were copied without further changes
from [hhrutter/tiff v1.0.6 testdata](https://github.com/hhrutter/tiff/tree/v1.0.6/testdata):

- `internal/pages/testdata/bw-gopher.png`
- `internal/pages/testdata/bw-gopher_ccittGroup3.tiff`
- `internal/pages/testdata/bw-gopher_ccittGroup4.tiff`

Retain this artwork attribution in addition to the upstream BSD notice.
These assets are used only for tests and are not embedded in the CLI binary.
No endorsement by the artist, the Go project, or dependency authors is implied.

The `synthetic-group{3,4}-photo{0,1}.tiff` files are project-generated geometric
test images, contain no gopher artwork, and are covered by the root MIT license.
Their generation with Pillow/libtiff does not add those tools to the CLI runtime.

## User-supplied assets

Stamp images and installed fonts are supplied by the user and are not relicensed
or bundled by imgstamp. Their respective licenses govern their use.
