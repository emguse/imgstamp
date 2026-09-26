# TIFF fixtures

`bw-gopher.png`, `bw-gopher_ccittGroup3.tiff`, and
`bw-gopher_ccittGroup4.tiff` are copied unchanged from
[github.com/hhrutter/tiff v1.0.6](https://github.com/hhrutter/tiff/tree/v1.0.6/testdata).
They exercise CCITT Group 3/4 decoding against known PNG pixels.
The upstream BSD-3-Clause notice for these copied fixtures is retained in
`LICENSE`. The Go gopher artwork was designed by **Renee French** and is
licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/), as
documented by the [Go project](https://go.dev/doc/gopher/README). The fixtures
are monochrome versions obtained from upstream, with no further changes here.
Keep this artwork attribution as well as the upstream BSD notice when
redistributing these files; the root MIT license does not replace them.

`synthetic-group{3,4}-photo{0,1}.tiff` are generated project fixtures (16 × 12
pixels): white everywhere except the black rectangle x=2..8, y=3..7.
They were generated with Pillow/libtiff using each compression and
PhotometricInterpretation value, then independently decoded with Pillow to
verify the expected pixels. They exercise the BlackIsZero CCITT correction
needed by hhrutter/tiff v1.0.6; the upstream gopher fixtures cover WhiteIsZero.
The synthetic fixtures contain no gopher artwork and are licensed under the
project's [MIT License](../../../LICENSE), not the copied fixture license.

See [Third-Party Notices](../../../THIRD_PARTY_NOTICES.md) for the complete
dependency inventory and distribution notices. Test fixtures are not embedded
in the CLI executable.
