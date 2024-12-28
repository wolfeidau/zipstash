# archive

This package contains the archive module which is used to create cache archives.

# inputs

The inputs to this module:

- `paths` a list of paths separated by new line, to add to the archive.
- `key` a key to use for the archive file name.
- `options` a list of options to pass to the archive library.

# outputs

The outputs from this module:

- `archive` the archive file.
- `sha256` the sha256 of the archive file.
- `size` the size of the archive file.
- `stats` the stats of the archive file.

# path handling

We have two types of paths.

* `absolute` paths are absolute paths to files or directories.
* `relative` paths are relative to the current working directory.
